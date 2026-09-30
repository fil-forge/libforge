// Package codec holds the CBOR and DAG-JSON primitives shared by the
// hand-written codecs in the commands packages. Those codecs exist for map
// types the cbor-gen / dag-json-gen generators cannot produce: DID- or
// CID-keyed maps and slice-valued maps.
package codec

import (
	"fmt"
	"io"
	"sort"

	jsg "github.com/alanshaw/dag-json-gen"
	"github.com/fil-forge/ucantone/did"
	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

// Size limits applied when encoding and decoding, matching the generators'
// defaults.
const (
	MaxString = 8192
	MaxLen    = 4096
)

// --- CBOR ---

// WriteCborString writes s as a CBOR text string.
func WriteCborString(cw *cbg.CborWriter, s string) error {
	if len(s) > MaxString {
		return xerrors.Errorf("string value was too long (%d)", len(s))
	}
	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(s))); err != nil {
		return err
	}
	_, err := cw.WriteString(s)
	return err
}

// ReadCborString reads a CBOR text string of at most [MaxString] bytes.
func ReadCborString(cr *cbg.CborReader) (string, error) {
	return cbg.ReadStringWithMax(cr, MaxString)
}

// WriteCborArrayHeader writes the header of a CBOR array with n elements.
func WriteCborArrayHeader(cw *cbg.CborWriter, n int) error {
	if n > MaxString {
		return xerrors.Errorf("slice value was too long (%d)", n)
	}
	return cw.WriteMajorTypeHeader(cbg.MajArray, uint64(n))
}

// ReadCborArrayHeader reads a CBOR array header and returns its length.
func ReadCborArrayHeader(cr *cbg.CborReader) (uint64, error) {
	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return 0, err
	}
	if maj != cbg.MajArray {
		return 0, fmt.Errorf("expected cbor array")
	}
	if extra > MaxString {
		return 0, fmt.Errorf("array too large (%d)", extra)
	}
	return extra, nil
}

// WriteCborMapHeader writes the header of a CBOR map with n entries.
func WriteCborMapHeader(cw *cbg.CborWriter, n int) error {
	if n > MaxLen {
		return xerrors.Errorf("map too large (%d)", n)
	}
	return cw.WriteMajorTypeHeader(cbg.MajMap, uint64(n))
}

// ReadCborMapHeader reads a CBOR map header and returns its entry count.
func ReadCborMapHeader(cr *cbg.CborReader) (uint64, error) {
	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return 0, err
	}
	if maj != cbg.MajMap {
		return 0, fmt.Errorf("expected a map (major type 5)")
	}
	if extra > MaxLen {
		return 0, fmt.Errorf("map too large (%d)", extra)
	}
	return extra, nil
}

// SortedDIDs returns the DID keys of m sorted by their string encoding, so the
// encoded map keys come out in the same (lexicographic) order the generators
// use for string-keyed maps.
func SortedDIDs[V any](m map[did.DID]V) []did.DID {
	keys := make([]did.DID, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	return keys
}

// SortedCIDs returns the CID keys of m sorted by their string encoding.
func SortedCIDs[V any](m map[cid.Cid]V) []cid.Cid {
	keys := make([]cid.Cid, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	return keys
}

// --- DAG-JSON ---

// WriteJSONKey writes an object key, preceded by a comma when comma is true
// (i.e. when it is not the first entry of the object).
func WriteJSONKey(jw *jsg.DagJsonWriter, name string, comma bool) error {
	if comma {
		if err := jw.WriteComma(); err != nil {
			return err
		}
	}
	if err := jw.WriteString(name); err != nil {
		return err
	}
	return jw.WriteObjectColon()
}

// ReadJSONObject reads a DAG-JSON object, invoking fn for each entry after
// consuming its key and colon. fn is responsible for reading the value.
func ReadJSONObject(jr *jsg.DagJsonReader, fn func(name string) error) (err error) {
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()
	if err := jr.ReadObjectOpen(); err != nil {
		return err
	}
	close, err := jr.PeekObjectClose()
	if err != nil {
		return err
	}
	if close {
		return jr.ReadObjectClose()
	}
	for i := 0; i < MaxString; i++ {
		name, err := jr.ReadString(MaxString)
		if err != nil {
			return err
		}
		if err := jr.ReadObjectColon(); err != nil {
			return err
		}
		if err := fn(name); err != nil {
			return err
		}
		close, err := jr.ReadObjectCloseOrComma()
		if err != nil {
			return err
		}
		if close {
			return nil
		}
	}
	return fmt.Errorf("map too large")
}

// ReadJSONArray reads a DAG-JSON array, invoking read for each element.
func ReadJSONArray[T any](jr *jsg.DagJsonReader, read func() (T, error)) ([]T, error) {
	if err := jr.ReadArrayOpen(); err != nil {
		return nil, err
	}
	close, err := jr.PeekArrayClose()
	if err != nil {
		return nil, err
	}
	var out []T
	if close {
		return out, jr.ReadArrayClose()
	}
	for {
		v, err := read()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
		close, err := jr.ReadArrayCloseOrComma()
		if err != nil {
			return nil, err
		}
		if close {
			return out, nil
		}
	}
}
