//go:build !codegen

package metrics

import (
	"io"

	jsg "github.com/alanshaw/dag-json-gen"
	"github.com/fil-forge/libforge/commands/internal/codec"
	"github.com/fil-forge/ucantone/did"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

// SampleSet has a hand-written codec because cbor-gen / dag-json-gen do not
// support DID-keyed maps. The CBOR and DAG-JSON primitives live in the shared
// internal codec package.

func (t SampleSet) MarshalCBOR(w io.Writer) error {
	cw := cbg.NewCborWriter(w)
	if err := codec.WriteCborMapHeader(cw, len(t.Entries)); err != nil {
		return err
	}
	for _, k := range codec.SortedDIDs(t.Entries) {
		if err := codec.WriteCborString(cw, k.String()); err != nil {
			return err
		}
		samples := t.Entries[k]
		if err := codec.WriteCborArrayHeader(cw, len(samples)); err != nil {
			return err
		}
		for i := range samples {
			if err := samples[i].MarshalCBOR(cw); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *SampleSet) UnmarshalCBOR(r io.Reader) error {
	cr := cbg.NewCborReader(r)
	n, err := codec.ReadCborMapHeader(cr)
	if err != nil {
		return err
	}
	m := make(map[did.DID][]SampleItem, n)
	for i := uint64(0); i < n; i++ {
		ks, err := codec.ReadCborString(cr)
		if err != nil {
			return err
		}
		k, err := did.Parse(ks)
		if err != nil {
			return xerrors.Errorf("parsing provider did %q: %w", ks, err)
		}
		sn, err := codec.ReadCborArrayHeader(cr)
		if err != nil {
			return err
		}
		samples := make([]SampleItem, sn)
		for j := uint64(0); j < sn; j++ {
			if err := samples[j].UnmarshalCBOR(cr); err != nil {
				return err
			}
		}
		m[k] = samples
	}
	*t = SampleSet{Entries: m}
	return nil
}

func (t SampleSet) MarshalDagJSON(w io.Writer) error {
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
		samples := t.Entries[k]
		for j := range samples {
			if j > 0 {
				if err := jw.WriteComma(); err != nil {
					return err
				}
			}
			if err := samples[j].MarshalDagJSON(jw); err != nil {
				return err
			}
		}
		if err := jw.WriteArrayClose(); err != nil {
			return err
		}
	}
	return jw.WriteObjectClose()
}

func (t *SampleSet) UnmarshalDagJSON(r io.Reader) error {
	jr := jsg.NewDagJsonReader(r)
	m := map[did.DID][]SampleItem{}
	err := codec.ReadJSONObject(jr, func(ks string) error {
		k, err := did.Parse(ks)
		if err != nil {
			return xerrors.Errorf("parsing provider did %q: %w", ks, err)
		}
		samples, err := codec.ReadJSONArray(jr, func() (SampleItem, error) {
			var s SampleItem
			if err := s.UnmarshalDagJSON(jr); err != nil {
				return SampleItem{}, err
			}
			return s, nil
		})
		if err != nil {
			return err
		}
		m[k] = samples
		return nil
	})
	if err != nil {
		return err
	}
	*t = SampleSet{Entries: m}
	return nil
}
