//go:build !codegen

// The codecs below call generated encoders, which the generator builds the
// package without.

package bucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	jsg "github.com/alanshaw/dag-json-gen"
	"github.com/fil-forge/libforge/commands/internal/codec"
	cbg "github.com/whyrusleeping/cbor-gen"
)

// canonical returns the statements deduplicated and ordered by
// [StatementCID]: the order every encoding writes. The result is never nil.
func (s StatementSet) canonical() []Statement {
	type keyed struct {
		key string
		st  Statement
	}
	keyed_ := make([]keyed, 0, len(s))
	for _, st := range s {
		keyed_ = append(keyed_, keyed{key: StatementCID(st).String(), st: st})
	}
	slices.SortFunc(keyed_, func(a, b keyed) int { return strings.Compare(a.key, b.key) })
	keyed_ = slices.CompactFunc(keyed_, func(a, b keyed) bool { return a.key == b.key })
	out := make([]Statement, len(keyed_))
	for i, k := range keyed_ {
		out[i] = k.st
	}
	return out
}

// MarshalJSON encodes the set as a list in canonical order; nil encodes as
// an empty list.
func (s StatementSet) MarshalJSON() ([]byte, error) { return json.Marshal(s.canonical()) }

// UnmarshalJSON decodes a list of statements, keeping the order given.
func (s *StatementSet) UnmarshalJSON(data []byte) error {
	var list []Statement
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&list); err != nil {
		return err
	}
	*s = list
	return nil
}

// MarshalCBOR writes the set as an array in canonical order.
func (s StatementSet) MarshalCBOR(w io.Writer) error {
	cw := cbg.NewCborWriter(w)
	c := s.canonical()
	if err := codec.WriteCborArrayHeader(cw, len(c)); err != nil {
		return err
	}
	for i := range c {
		if err := c[i].MarshalCBOR(cw); err != nil {
			return err
		}
	}
	return nil
}

// UnmarshalCBOR reads an array of statements, at most [MaxSetSize].
func (s *StatementSet) UnmarshalCBOR(r io.Reader) error {
	cr := cbg.NewCborReader(r)
	n, err := codec.ReadCborArrayHeader(cr)
	if err != nil {
		return err
	}
	if n > MaxSetSize {
		return fmt.Errorf("statement set holds %d statements, at most %d are allowed", n, MaxSetSize)
	}
	out := make(StatementSet, n)
	for i := range out {
		if err := out[i].UnmarshalCBOR(cr); err != nil {
			return err
		}
	}
	*s = out
	return nil
}

// MarshalDagJSON writes the set as an array in canonical order.
func (s StatementSet) MarshalDagJSON(w io.Writer) error {
	jw := jsg.NewDagJsonWriter(w)
	if err := jw.WriteArrayOpen(); err != nil {
		return err
	}
	for i, st := range s.canonical() {
		if i > 0 {
			if err := jw.WriteComma(); err != nil {
				return err
			}
		}
		if err := st.MarshalDagJSON(jw); err != nil {
			return err
		}
	}
	return jw.WriteArrayClose()
}

// UnmarshalDagJSON reads an array of statements, at most [MaxSetSize].
func (s *StatementSet) UnmarshalDagJSON(r io.Reader) error {
	jr := jsg.NewDagJsonReader(r)
	list, err := codec.ReadJSONArray(jr, func() (Statement, error) {
		var st Statement
		err := st.UnmarshalDagJSON(jr)
		return st, err
	})
	if err != nil {
		return err
	}
	if len(list) > MaxSetSize {
		return fmt.Errorf("statement set holds %d statements, at most %d are allowed", len(list), MaxSetSize)
	}
	*s = list
	return nil
}

// The document's own codecs are hand-written because the generators inline a
// slice field by its kind and would bypass StatementSet's canonical order.
// They write the layout the generators would: CBOR keys shortest first,
// DAG-JSON keys sorted.

// MarshalCBOR writes d as a map of version and statement.
func (d *PolicyDocument) MarshalCBOR(w io.Writer) error {
	cw := cbg.NewCborWriter(w)
	if d == nil {
		_, err := cw.Write(cbg.CborNull)
		return err
	}
	if err := codec.WriteCborMapHeader(cw, 2); err != nil {
		return err
	}
	if err := codec.WriteCborString(cw, "version"); err != nil {
		return err
	}
	if err := codec.WriteCborString(cw, d.Version); err != nil {
		return err
	}
	if err := codec.WriteCborString(cw, "statement"); err != nil {
		return err
	}
	return d.Statements.MarshalCBOR(cw)
}

// UnmarshalCBOR reads a map of version and statement.
func (d *PolicyDocument) UnmarshalCBOR(r io.Reader) error {
	*d = PolicyDocument{}
	cr := cbg.NewCborReader(r)
	n, err := codec.ReadCborMapHeader(cr)
	if err != nil {
		return err
	}
	for i := uint64(0); i < n; i++ {
		key, err := codec.ReadCborString(cr)
		if err != nil {
			return err
		}
		switch key {
		case "version":
			d.Version, err = codec.ReadCborString(cr)
		case "statement":
			err = d.Statements.UnmarshalCBOR(cr)
		default:
			return fmt.Errorf("unknown field %q in a policy document", key)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// MarshalDagJSON writes d as an object of statement and version.
func (d *PolicyDocument) MarshalDagJSON(w io.Writer) error {
	jw := jsg.NewDagJsonWriter(w)
	if d == nil {
		return jw.WriteNull()
	}
	if err := jw.WriteObjectOpen(); err != nil {
		return err
	}
	if err := codec.WriteJSONKey(jw, "statement", false); err != nil {
		return err
	}
	if err := d.Statements.MarshalDagJSON(jw); err != nil {
		return err
	}
	if err := codec.WriteJSONKey(jw, "version", true); err != nil {
		return err
	}
	if err := jw.WriteString(d.Version); err != nil {
		return err
	}
	return jw.WriteObjectClose()
}

// UnmarshalDagJSON reads an object of statement and version.
func (d *PolicyDocument) UnmarshalDagJSON(r io.Reader) error {
	*d = PolicyDocument{}
	jr := jsg.NewDagJsonReader(r)
	return codec.ReadJSONObject(jr, func(name string) error {
		switch name {
		case "version":
			v, err := jr.ReadString(codec.MaxString)
			d.Version = v
			return err
		case "statement":
			return d.Statements.UnmarshalDagJSON(jr)
		default:
			return fmt.Errorf("unknown field %q in a policy document", name)
		}
	})
}
