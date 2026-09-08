//go:build !codegen

package s3

import (
	"io"

	jsg "github.com/alanshaw/dag-json-gen"
	"github.com/fil-forge/libforge/commands/internal/codec"
	"github.com/fil-forge/ucantone/did"
	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

// These codecs are hand-written because cbor-gen / dag-json-gen do not support
// slice-valued maps. The CBOR and DAG-JSON primitives live in the shared
// internal codec package.

// --- PermissionSet: map[did.DID][]string ---

func (t PermissionSet) MarshalCBOR(w io.Writer) error {
	cw := cbg.NewCborWriter(w)
	if err := codec.WriteCborMapHeader(cw, len(t.Entries)); err != nil {
		return err
	}
	for _, k := range codec.SortedDIDs(t.Entries) {
		if err := codec.WriteCborString(cw, k.String()); err != nil {
			return err
		}
		perms := t.Entries[k]
		if err := codec.WriteCborArrayHeader(cw, len(perms)); err != nil {
			return err
		}
		for _, p := range perms {
			if err := codec.WriteCborString(cw, p); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *PermissionSet) UnmarshalCBOR(r io.Reader) error {
	cr := cbg.NewCborReader(r)
	n, err := codec.ReadCborMapHeader(cr)
	if err != nil {
		return err
	}
	m := make(map[did.DID][]string, n)
	for i := uint64(0); i < n; i++ {
		ks, err := codec.ReadCborString(cr)
		if err != nil {
			return err
		}
		k, err := did.Parse(ks)
		if err != nil {
			return xerrors.Errorf("parsing access key did %q: %w", ks, err)
		}
		alen, err := codec.ReadCborArrayHeader(cr)
		if err != nil {
			return err
		}
		perms := make([]string, 0, alen)
		for j := uint64(0); j < alen; j++ {
			p, err := codec.ReadCborString(cr)
			if err != nil {
				return err
			}
			perms = append(perms, p)
		}
		m[k] = perms
	}
	*t = PermissionSet{Entries: m}
	return nil
}

func (t PermissionSet) MarshalDagJSON(w io.Writer) error {
	jw := jsg.NewDagJsonWriter(w)
	if err := jw.WriteObjectOpen(); err != nil {
		return err
	}
	for i, k := range codec.SortedDIDs(t.Entries) {
		if err := codec.WriteJSONKey(jw, k.String(), i > 0); err != nil {
			return err
		}
		if err := jw.WriteArrayOpen(); err != nil {
			return err
		}
		for j, p := range t.Entries[k] {
			if j > 0 {
				if err := jw.WriteComma(); err != nil {
					return err
				}
			}
			if err := jw.WriteString(p); err != nil {
				return err
			}
		}
		if err := jw.WriteArrayClose(); err != nil {
			return err
		}
	}
	return jw.WriteObjectClose()
}

func (t *PermissionSet) UnmarshalDagJSON(r io.Reader) error {
	jr := jsg.NewDagJsonReader(r)
	m := map[did.DID][]string{}
	err := codec.ReadJSONObject(jr, func(ks string) error {
		k, err := did.Parse(ks)
		if err != nil {
			return xerrors.Errorf("parsing access key did %q: %w", ks, err)
		}
		perms, err := codec.ReadJSONArray(jr, func() (string, error) { return jr.ReadString(codec.MaxString) })
		if err != nil {
			return err
		}
		m[k] = perms
		return nil
	})
	if err != nil {
		return err
	}
	*t = PermissionSet{Entries: m}
	return nil
}

// --- ProofSet: map[cid.Cid][]cid.Cid ---

func (t ProofSet) MarshalCBOR(w io.Writer) error {
	cw := cbg.NewCborWriter(w)
	if err := codec.WriteCborMapHeader(cw, len(t.Entries)); err != nil {
		return err
	}
	for _, k := range codec.SortedCIDs(t.Entries) {
		if err := codec.WriteCborString(cw, k.String()); err != nil {
			return err
		}
		links := t.Entries[k]
		if err := codec.WriteCborArrayHeader(cw, len(links)); err != nil {
			return err
		}
		for _, c := range links {
			if err := cbg.WriteCid(cw, c); err != nil {
				return xerrors.Errorf("failed to write cid: %w", err)
			}
		}
	}
	return nil
}

func (t *ProofSet) UnmarshalCBOR(r io.Reader) error {
	cr := cbg.NewCborReader(r)
	n, err := codec.ReadCborMapHeader(cr)
	if err != nil {
		return err
	}
	m := make(map[cid.Cid][]cid.Cid, n)
	for i := uint64(0); i < n; i++ {
		ks, err := codec.ReadCborString(cr)
		if err != nil {
			return err
		}
		k, err := cid.Decode(ks)
		if err != nil {
			return xerrors.Errorf("parsing delegation cid %q: %w", ks, err)
		}
		alen, err := codec.ReadCborArrayHeader(cr)
		if err != nil {
			return err
		}
		links := make([]cid.Cid, 0, alen)
		for j := uint64(0); j < alen; j++ {
			c, err := cbg.ReadCid(cr)
			if err != nil {
				return xerrors.Errorf("failed to read cid: %w", err)
			}
			links = append(links, c)
		}
		m[k] = links
	}
	*t = ProofSet{Entries: m}
	return nil
}

func (t ProofSet) MarshalDagJSON(w io.Writer) error {
	jw := jsg.NewDagJsonWriter(w)
	if err := jw.WriteObjectOpen(); err != nil {
		return err
	}
	for i, k := range codec.SortedCIDs(t.Entries) {
		if err := codec.WriteJSONKey(jw, k.String(), i > 0); err != nil {
			return err
		}
		if err := jw.WriteArrayOpen(); err != nil {
			return err
		}
		for j, c := range t.Entries[k] {
			if j > 0 {
				if err := jw.WriteComma(); err != nil {
					return err
				}
			}
			if err := jw.WriteCid(c); err != nil {
				return err
			}
		}
		if err := jw.WriteArrayClose(); err != nil {
			return err
		}
	}
	return jw.WriteObjectClose()
}

func (t *ProofSet) UnmarshalDagJSON(r io.Reader) error {
	jr := jsg.NewDagJsonReader(r)
	m := map[cid.Cid][]cid.Cid{}
	err := codec.ReadJSONObject(jr, func(ks string) error {
		k, err := cid.Decode(ks)
		if err != nil {
			return xerrors.Errorf("parsing delegation cid %q: %w", ks, err)
		}
		links, err := codec.ReadJSONArray(jr, func() (cid.Cid, error) { return jr.ReadCid() })
		if err != nil {
			return err
		}
		m[k] = links
		return nil
	})
	if err != nil {
		return err
	}
	*t = ProofSet{Entries: m}
	return nil
}

// --- KeySet: map[did.DID][]VerificationKey ---

func (t KeySet) MarshalCBOR(w io.Writer) error {
	cw := cbg.NewCborWriter(w)
	if err := codec.WriteCborMapHeader(cw, len(t.Entries)); err != nil {
		return err
	}
	for _, k := range codec.SortedDIDs(t.Entries) {
		if err := codec.WriteCborString(cw, k.String()); err != nil {
			return err
		}
		keys := t.Entries[k]
		if err := codec.WriteCborArrayHeader(cw, len(keys)); err != nil {
			return err
		}
		for i := range keys {
			if err := keys[i].MarshalCBOR(cw); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *KeySet) UnmarshalCBOR(r io.Reader) error {
	cr := cbg.NewCborReader(r)
	n, err := codec.ReadCborMapHeader(cr)
	if err != nil {
		return err
	}
	m := make(map[did.DID][]VerificationKey, n)
	for i := uint64(0); i < n; i++ {
		ks, err := codec.ReadCborString(cr)
		if err != nil {
			return err
		}
		k, err := did.Parse(ks)
		if err != nil {
			return xerrors.Errorf("parsing access key did %q: %w", ks, err)
		}
		alen, err := codec.ReadCborArrayHeader(cr)
		if err != nil {
			return err
		}
		keys := make([]VerificationKey, alen)
		for j := uint64(0); j < alen; j++ {
			if err := keys[j].UnmarshalCBOR(cr); err != nil {
				return err
			}
		}
		m[k] = keys
	}
	*t = KeySet{Entries: m}
	return nil
}

func (t KeySet) MarshalDagJSON(w io.Writer) error {
	jw := jsg.NewDagJsonWriter(w)
	if err := jw.WriteObjectOpen(); err != nil {
		return err
	}
	for i, k := range codec.SortedDIDs(t.Entries) {
		if err := codec.WriteJSONKey(jw, k.String(), i > 0); err != nil {
			return err
		}
		if err := jw.WriteArrayOpen(); err != nil {
			return err
		}
		keys := t.Entries[k]
		for j := range keys {
			if j > 0 {
				if err := jw.WriteComma(); err != nil {
					return err
				}
			}
			if err := keys[j].MarshalDagJSON(jw); err != nil {
				return err
			}
		}
		if err := jw.WriteArrayClose(); err != nil {
			return err
		}
	}
	return jw.WriteObjectClose()
}

func (t *KeySet) UnmarshalDagJSON(r io.Reader) error {
	jr := jsg.NewDagJsonReader(r)
	m := map[did.DID][]VerificationKey{}
	err := codec.ReadJSONObject(jr, func(ks string) error {
		k, err := did.Parse(ks)
		if err != nil {
			return xerrors.Errorf("parsing access key did %q: %w", ks, err)
		}
		keys, err := codec.ReadJSONArray(jr, func() (VerificationKey, error) {
			var vk VerificationKey
			err := vk.UnmarshalDagJSON(jr)
			return vk, err
		})
		if err != nil {
			return err
		}
		m[k] = keys
		return nil
	})
	if err != nil {
		return err
	}
	*t = KeySet{Entries: m}
	return nil
}
