//go:build !codegen

package routing

import (
	"io"

	jsg "github.com/alanshaw/dag-json-gen"
	"github.com/fil-forge/libforge/commands/internal/codec"
	"github.com/fil-forge/ucantone/did"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

// CandidateSet has a hand-written codec because cbor-gen / dag-json-gen do not
// support DID-keyed maps. The CBOR and DAG-JSON primitives live in the shared
// internal codec package.

func (t CandidateSet) MarshalCBOR(w io.Writer) error {
	cw := cbg.NewCborWriter(w)
	if err := codec.WriteCborMapHeader(cw, len(t.Entries)); err != nil {
		return err
	}
	for _, k := range codec.SortedDIDs(t.Entries) {
		if err := codec.WriteCborString(cw, k.String()); err != nil {
			return err
		}
		c := t.Entries[k]
		if err := c.MarshalCBOR(cw); err != nil {
			return err
		}
	}
	return nil
}

func (t *CandidateSet) UnmarshalCBOR(r io.Reader) error {
	cr := cbg.NewCborReader(r)
	n, err := codec.ReadCborMapHeader(cr)
	if err != nil {
		return err
	}
	m := make(map[did.DID]Candidate, n)
	for i := uint64(0); i < n; i++ {
		ks, err := codec.ReadCborString(cr)
		if err != nil {
			return err
		}
		k, err := did.Parse(ks)
		if err != nil {
			return xerrors.Errorf("parsing candidate did %q: %w", ks, err)
		}
		var c Candidate
		if err := c.UnmarshalCBOR(cr); err != nil {
			return err
		}
		m[k] = c
	}
	*t = CandidateSet{Entries: m}
	return nil
}

func (t CandidateSet) MarshalDagJSON(w io.Writer) error {
	jw := jsg.NewDagJsonWriter(w)
	if err := jw.WriteObjectOpen(); err != nil {
		return err
	}
	for i, k := range codec.SortedDIDs(t.Entries) {
		if err := codec.WriteJSONKey(jw, k.String(), i > 0); err != nil {
			return err
		}
		c := t.Entries[k]
		if err := c.MarshalDagJSON(jw); err != nil {
			return err
		}
	}
	return jw.WriteObjectClose()
}

func (t *CandidateSet) UnmarshalDagJSON(r io.Reader) error {
	jr := jsg.NewDagJsonReader(r)
	m := map[did.DID]Candidate{}
	err := codec.ReadJSONObject(jr, func(ks string) error {
		k, err := did.Parse(ks)
		if err != nil {
			return xerrors.Errorf("parsing candidate did %q: %w", ks, err)
		}
		var c Candidate
		if err := c.UnmarshalDagJSON(jr); err != nil {
			return err
		}
		m[k] = c
		return nil
	})
	if err != nil {
		return err
	}
	*t = CandidateSet{Entries: m}
	return nil
}
