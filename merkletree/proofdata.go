// Package merkletree is a thin libforge wrapper around
// go-data-segment's merkletree.ProofData that adds DagJSON codec methods.
//
// The merkletree algorithm itself is not reimplemented here; conversions
// to/from the canonical go-data-segment type are zero-cost since the
// underlying struct layout is identical.
package merkletree

import (
	"fmt"
	"io"

	jsg "github.com/alanshaw/dag-json-gen"
	dsmerkle "github.com/filecoin-project/go-data-segment/merkletree"
)

// Node is re-exported from go-data-segment for ergonomic use alongside
// the wrapped ProofData.
type Node = dsmerkle.Node

// NodeSize matches go-data-segment's NodeSize.
const NodeSize = dsmerkle.NodeSize

// ProofData is a merkle inclusion proof, layout-identical to
// go-data-segment's merkletree.ProofData. Use a named-type conversion to
// move between the two:
//
//	libforgePD := merkletree.ProofData(dsPD)
//	dsPD       := dsmerkle.ProofData(libforgePD)
type ProofData dsmerkle.ProofData

// MarshalCBOR delegates to go-data-segment's CBOR codec via a zero-cost
// named-type conversion.
func (p *ProofData) MarshalCBOR(w io.Writer) error {
	pd := dsmerkle.ProofData(*p)
	return pd.MarshalCBOR(w)
}

// UnmarshalCBOR delegates to go-data-segment's CBOR codec.
func (p *ProofData) UnmarshalCBOR(r io.Reader) error {
	var pd dsmerkle.ProofData
	if err := pd.UnmarshalCBOR(r); err != nil {
		return err
	}
	*p = ProofData(pd)
	return nil
}

// MarshalDagJSON writes the proof as a DagJSON map: {"path": [<bytes>...], "index": N}.
// Path nodes use DagJSON's canonical bytes encoding ({"/": {"bytes": "<base64>"}}).
func (p *ProofData) MarshalDagJSON(w io.Writer) error {
	jw := jsg.NewDagJsonWriter(w)

	if err := jw.WriteObjectOpen(); err != nil {
		return err
	}

	if err := jw.WriteString("path"); err != nil {
		return err
	}
	if err := jw.WriteObjectColon(); err != nil {
		return err
	}
	if err := jw.WriteArrayOpen(); err != nil {
		return err
	}
	for i, n := range p.Path {
		if i > 0 {
			if err := jw.WriteComma(); err != nil {
				return err
			}
		}
		if err := jw.WriteBytes(n[:]); err != nil {
			return err
		}
	}
	if err := jw.WriteArrayClose(); err != nil {
		return err
	}

	if err := jw.WriteComma(); err != nil {
		return err
	}

	if err := jw.WriteString("index"); err != nil {
		return err
	}
	if err := jw.WriteObjectColon(); err != nil {
		return err
	}
	if err := jw.WriteUint64(p.Index); err != nil {
		return err
	}

	return jw.WriteObjectClose()
}

// UnmarshalDagJSON reads the proof from the DagJSON form produced by MarshalDagJSON.
func (p *ProofData) UnmarshalDagJSON(r io.Reader) error {
	jr := jsg.NewDagJsonReader(r)
	*p = ProofData{}

	if err := jr.ReadObjectOpen(); err != nil {
		return err
	}
	empty, err := jr.PeekObjectClose()
	if err != nil {
		return err
	}
	if empty {
		return jr.ReadObjectClose()
	}
	for {
		key, err := jr.ReadString(8192)
		if err != nil {
			return err
		}
		if err := jr.ReadObjectColon(); err != nil {
			return err
		}
		switch key {
		case "path":
			if err := jr.ReadArrayOpen(); err != nil {
				return err
			}
			emptyArr, err := jr.PeekArrayClose()
			if err != nil {
				return err
			}
			if emptyArr {
				if err := jr.ReadArrayClose(); err != nil {
					return err
				}
			} else {
				for {
					b, err := jr.ReadBytes(NodeSize)
					if err != nil {
						return fmt.Errorf("reading path node %d: %w", len(p.Path), err)
					}
					if len(b) != NodeSize {
						return fmt.Errorf("path node %d size %d, want %d", len(p.Path), len(b), NodeSize)
					}
					var n Node
					copy(n[:], b)
					p.Path = append(p.Path, n)

					done, err := jr.ReadArrayCloseOrComma()
					if err != nil {
						return err
					}
					if done {
						break
					}
				}
			}
		case "index":
			n, err := jr.ReadNumberAsUint64()
			if err != nil {
				return err
			}
			p.Index = n
		default:
			return fmt.Errorf("unexpected field %q", key)
		}

		done, err := jr.ReadObjectCloseOrComma()
		if err != nil {
			return err
		}
		if done {
			break
		}
	}
	return nil
}
