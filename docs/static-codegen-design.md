# Design: Static, annotation-driven serialization codegen

Status: proposal
Author: (design doc — generated for review)
Scope: replace the reflection-based `cbor-gen` / `dag-json-gen` codegen across
`libforge` with a single static tool that selects types via annotations and
never compiles or executes the target packages.

---

## 1. Motivation

Today every serialized type is encoded/decoded by code emitted from two
**reflection-based** generators:

- `github.com/whyrusleeping/cbor-gen` (v0.3.1)
- `github.com/alanshaw/dag-json-gen` (v0.0.6)

Each generator lives in its own `package main` under a `gen/` directory (20 of
them), imports its sibling package, constructs *instances* of the target structs
(`blob.Blob{}`, `blob.AddOK{}`, …), and hands those values to
`cbg.WriteMapEncodersToFile` / `jsg.Write*EncodersToFile`, which walk them with
`reflect` and read the `cborgen:` / `dagjsongen:` struct tags.

This has three concrete costs:

1. **Bootstrap deadlock.** Reflecting over `blob.Blob{}` requires importing the
   `blob` package, so `blob` and its whole transitive graph must *compile*
   before codegen runs. When the thing blocking compilation is the missing
   generated code (e.g. hand-written code calls `T.MarshalCBOR` before it
   exists), you cannot regenerate your way out.
2. **Package littering.** 20 near-identical `gen/main.go` files, each
   re-copying the same `tag()` helper and model list.
3. **A fragile, inconsistent workaround.** Generated files carry
   `//go:build !codegen` and generators run with `-tags codegen` so stale output
   drops out of the build. This only decouples the *generated file*, not a
   genuine compile error in hand-written source, and not the case where
   hand-written code depends on a generated method. It is also applied
   inconsistently — `blobindex/datamodel` uses neither the build tag nor
   `-tags codegen`.

The reflection requirement is intrinsic to these libraries; it cannot be made
static without replacing the front end. The modern idiom (`stringer`,
`controller-gen`, `deepcopy-gen`) is a single tool that parses source with
`go/types` and selects types via a marker comment — which never compiles or runs
the target package, and therefore eliminates problem (1) entirely.

## 2. Goals / non-goals

**Goals**

- One tool (`cmd/forgegen`), one `//go:generate` directive, zero `gen/`
  packages.
- Type selection by annotation (a doc-comment directive), not by hand-curated
  `[]any{}` slices.
- Generation works even when the target package does not compile.
- Drop the `codegen` build tag and the `tag()` helper entirely.
- **Byte-for-byte identical output** to today (see §3).

**Non-goals**

- Changing the CBOR / dag-json wire formats.
- Changing the field-tag vocabulary (`cborgen:` / `dagjsongen:` stay).
- A general-purpose IPLD schema compiler. This replaces the existing two
  generators and nothing more.

## 3. Compatibility decision: keep the bytes frozen

The formats are not shipped but are deep in development. **Recommendation: keep
output byte-identical**, gated by golden tests. I looked for a reason to *want*
the bytes to change and did not find a good one:

- cbor-gen's encoding (deterministic map-key ordering, major-type headers,
  pointer/`omitempty` nil handling) is well-tested and correct. There is no
  encoding defect that a rewrite would fix.
- The only cbor-gen behaviors one might want to revisit are **validation
  limits**, not encodings — e.g. its hard-coded `ByteArrayMaxLen` / `MaxLength`
  read-side caps. Those affect what input is *accepted*, not the bytes produced
  for valid data, and can be revisited independently of this rewrite.

So: freeze the bytes. If a future type genuinely needs a different encoding,
that's a deliberate, reviewed format change — orthogonal to this migration.

This decision drives the whole architecture (§4): we want the lowest-risk path
to identical bytes, which means **reusing the existing emitters' byte-producing
code unchanged** and replacing only how they learn about types.

## 4. Architecture

Three stages: **discover → model → emit.**

```
go/packages (types, syntax)      annotation scan        emit (ported cbor-gen /
   tolerant of compile errors  →   + field tags      →   dag-json-gen back end)
```

### 4.1 Discovery & type model — `go/types`, not raw AST

Load packages with `golang.org/x/tools/go/packages` in a mode that includes
`NeedTypes | NeedTypesInfo | NeedSyntax | NeedImports | NeedDeps`. Crucially, we
**do not fail on `pkg.Errors`**. The go/types checker is error-tolerant: it
returns a best-effort `*types.Package` even when some files don't type-check,
inserting `Invalid` only for the parts it can't resolve. As long as the specific
annotated type and its fields resolve, we can generate — which is exactly the
bootstrap case we need to support.

> Honest caveat: this is *best-effort tolerant*, not magic. If the annotated
> struct's own fields reference unresolved types, generation for that type
> fails with a clear diagnostic. That's strictly better than today (where any
> error anywhere blocks everything), but it is not "compiles arbitrary garbage."

Raw `go/ast` (syntax only) would be even more tolerant, but `libforge`'s types
lean heavily on cross-package **named** types — `did.DID`, `cid.Cid`,
`multihash.Multihash`, `promise.AwaitOK`, `commands.CborURL`, `big.Int`,
`merkletree.ProofData`, `ucan.Command` — many of which are named types over
`[]byte`/`string` or carry their own `MarshalCBOR`. Resolving those (underlying
type? implements the marshaler interface?) from bare AST means reimplementing a
type resolver. `go/types` gives us that for free. We use it; we just tolerate
its errors.

### 4.2 Annotation scheme

Type selection moves from the curated slices in `gen/main.go` to a **doc-comment
directive** on the type:

```go
//forge:codegen cbor=map,dagjson=map
type Blob struct { … }

//forge:codegen cbor=tuple,dagjson=tuple
type RangeModel struct { … }
```

- The directive captures what the slice-name distinction captures today: the
  **map vs tuple** encoding choice (cf. `blobindex/datamodel/gen`, which splits
  `mapModels` from `tupleModels`).
- It also captures **which encoders** to emit (`cbor`, `dagjson`, or both), so a
  type can opt into one and not the other.
- Output file + package name are derived from the type's own package, removing
  the hard-coded `"../cbor_gen.go"` / package-name strings.

Field-level tags (`cborgen:"…"`, `dagjsongen:"…"`) are unchanged.

Note on "build tags": a `//go:build` constraint is **file-level**, so it can't
mark an individual *type* for codegen — hence a doc-comment directive rather than
a build tag for selection. The one build tag in play today (`//go:build
!codegen` on generated output) **goes away**: a static tool never compiles the
target package, so there is nothing to tag out, no `-tags codegen`, no `tag()`
helper, and no `codegen-build` Makefile target.

### 4.3 Emission — fork-and-reskin, don't reimplement

This is the make-or-break for "frozen bytes," and the key design decision.

The naive plan — reimplement the encoders from scratch against `go/types` —
maximizes the risk of subtle byte drift across ~19.4k lines of generated code
(key ordering, header bytes, `omitempty` semantics, scratch-buffer use, the
`// t.Field (type)` comments cbor-gen emits, interface detection). We reject it.

Instead: **keep cbor-gen's and dag-json-gen's byte-producing code verbatim and
replace only how it learns about types.** Internally each generator is two
layers, and — verified against `whyrusleeping/cbor-gen@v0.3.1` — **both are
already exported**:

```
ParseTypeInfo(interface{}) (*GenTypeInfo, error)               ← reflect front end (gen.go:239)
GenMapEncodersForType / GenTupleEncodersForType(*GenTypeInfo)   ← byte-producing back end (gen.go:1997 / :1698)
type GenTypeInfo struct { Fields []Field }                     ← gen.go:199
type Field struct { Type reflect.Type; … }                     ← gen.go:141
```

The back end is where the bytes come from. If we leave it untouched and supply a
`GenTypeInfo` built from `go/types` instead of `reflect`, output is identical by
construction.

**So why a fork at all, if the back end is exported?** Because of exactly one
field: `Field.Type` is a concrete `reflect.Type`, and the back end is hard-wired
to it — in `v0.3.1`'s `gen.go` that's **~83 reflect-method calls** on field
types (`43× Kind`, `26× Elem`, plus `Key`/`Len`/`Name`/`PkgPath`/`String`/
`NumField`) and **3 identity sentinels** built with `reflect.TypeOf` and compared
by `==`:

```go
cidType      = reflect.TypeOf(cid.Cid{})
bigIntType   = reflect.TypeOf(big.Int{})
deferredType = reflect.TypeOf(Deferred{})
// … if f.Type == cidType { … }
```

You cannot populate `Field.Type` from `go/types`, and you cannot faithfully
synthesize a `reflect.Type` (named types, custom-marshaler detection, recursion,
and those `==` checks all need the real runtime type). So the export lets us
*call* the emitter but not *feed it without reflection*. Removing the reflection
requirement means editing `Field.Type`'s type and the sentinel comparisons —
source changes, hence a patched copy.

**"Fork" = a patched copy we own in-tree** (e.g. `cmd/forgegen/internal/cborgen`),
*not* a GitHub fork. (A GitHub fork + `go.mod replace` is possible but adds no
value here.) The patch is small and mechanical: introduce an interface — call it
`TypeRef` — replacing `reflect.Type` in `Field.Type` and the few helpers, and
turn the 3 sentinels into methods (`IsCid()` / `IsBigInt()` / `IsDeferred()`).
`Kind()` can keep returning `reflect.Kind` (it's just an enum — only
`reflect.Type` itself is unsynthesizable), so the 43 `.Kind()` switch cases stay
untouched.

```go
type TypeRef interface {
    Kind() reflect.Kind   // reuse the enum; only reflect.Type is unsynthesizable
    Elem() TypeRef
    Key() TypeRef
    Len() int
    Name() string
    PkgPath() string
    String() string
    Fields() []FieldRef    // nested struct literals
    Implements(iface) bool // CBORMarshaler / json marshaler detection
    IsCid() bool; IsBigInt() bool; IsDeferred() bool
}
```

Provide two implementations: a `reflect`-backed one (lets us A/B against upstream
during the port) and the real `go/types`-backed one. The ~2000 lines of
byte-producing logic are otherwise unchanged — which is what keeps the output
frozen. Divergence from upstream is confined to the `TypeRef` seam.

> Considered and rejected: transpiling the parsed structs into a throwaway
> package, compiling *that* in isolation, and reflecting over it (avoids the
> fork). It fails because the synthetic package still imports the real referenced
> types (`did.DID`, `promise.AwaitOK`, …), so those must compile — back to the
> bootstrap requirement.

### 4.4 The tool — `cmd/forgegen`

```
cmd/forgegen/
  main.go         // load module packages, scan for //forge:codegen, group, emit
  discover.go     // go/packages load (error-tolerant) + directive parsing
  typeref/        // go/types-backed TypeRef
  internal/cborgen, internal/dagjsongen   // forked back ends + reflect TypeRef
```

Root directive: a single `//go:generate go run ./cmd/forgegen ./...` (or have
the tool default to scanning the whole module). Output files keep their current
names and locations so the diff is purely the *contents* (which should be empty
once parity is reached — see §5).

## 5. Migration & golden testing

The repo already has the acceptance test for this work: `make gen-check`
regenerates and fails if any `*_gen.go` / `*_gen.*.go` changed. That becomes the
parity gate for the rewrite.

Process:

1. Land `cmd/forgegen` alongside the existing generators (don't delete anything
   yet). Add the `//forge:codegen` directives to the target types.
2. Point `forgegen` at **one** package (start with `blobindex/datamodel`: it's
   small and exercises both map *and* tuple encoders). Run it, `git diff` the
   output — iterate on the fork until the diff is **empty**.
3. Expand package by package. Each package is "done" when `forgegen` produces a
   zero diff against the committed cbor-gen/dag-json-gen output.
4. When all packages reach zero-diff: delete the 20 `gen/` packages, drop the
   `codegen` build tag from generated files, remove the `tag()` helper and the
   `codegen-build` Makefile target, and switch the root `//go:generate`.
5. CI's `gen-check` now guards the new tool exactly as it guarded the old ones.

Because step 2–3's success criterion is *byte-identical output*, the migration
is verifiable at every step and reversible until the final cutover.

## 6. Risks & open questions

| Risk | Mitigation |
|---|---|
| Subtle byte drift vs. upstream emitters | Fork-and-reskin (§4.3) keeps the byte-producing code unchanged; `gen-check` enforces zero diff per package. |
| `go/types` can't resolve an annotated type in a broken package | Generation for that type fails with a clear error — still strictly better than today; the common bootstrap case (missing generated methods) resolves fine. |
| Owning a fork of two upstream libs | Seam is limited to the `TypeRef` front end; pin upstream versions; the back ends are effectively frozen anyway (we depend on their exact output). |
| Interface detection (custom `MarshalCBOR`, `cbg.CBORMarshaler`) under go/types | Implement `TypeRef.Implements` via `types.Implements` against the marshaler interface types; covered by the datamodel parity test, which includes types with custom marshalers. |
| Directive drift / typos silently skip a type | `forgegen` errors on an unrecognized `//forge:codegen` key and can optionally warn on exported serialized-looking types with field tags but no directive. |

**Open questions for review:**

- Directive syntax — `//forge:codegen cbor=map,dagjson=map` as proposed, or a
  terser default (`//forge:codegen` ⇒ both encoders, map) with overrides only
  when needed?
- Should `forgegen` live in this module (`./cmd/forgegen`) or a separate module
  to keep `x/tools` out of the main module's dependency graph? (Separate module
  avoids adding `golang.org/x/tools` — not currently a dependency — to consumers
  of `libforge`.)
- Do we want the optional "serialized-looking type without a directive" lint, or
  is that too noisy?

## 7. Effort estimate

- `cmd/forgegen` discovery + directive parsing + go/types loading: **small.**
- The fork's `TypeRef` seam + go/types-backed implementation for cbor-gen:
  **medium** — the bulk of the work, and where parity is won or lost.
- Same for dag-json-gen: **small–medium** (same pattern, second time).
- Per-package parity bring-up + final cutover: **medium**, mostly iteration
  against `gen-check`.

Net: a contained project whose hard part is well-isolated (the `TypeRef` seam)
and whose correctness is mechanically checkable at every step (zero-diff
parity). The risk profile is low *because* we keep the emitters' bytes and only
swap their source of type information.
