//go:build !codegen

package blob

import (
	"fmt"
	"io"

	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/errors"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

// UnsupportedDigestCodeErrorName is the stable receipt-failure name when a
// `/blob/add` or `/blob/allocate` names a digest code the executor does not
// support, or the executor does not support adding a blob by digest code at
// all. A client that receives it can fall back to computing the digest before
// adding the blob.
const UnsupportedDigestCodeErrorName = "UnsupportedDigestCode"

// ErrUnsupportedDigestCode is the failure for an unsupported digest code.
var ErrUnsupportedDigestCode = errors.New(UnsupportedDigestCodeErrorName, "adding a blob by digest code is supported only for sha2-256")

var (
	errNoVariant    = errors.New("InvalidUnion", "union holds no variant")
	errBothVariants = errors.New("InvalidUnion", "union holds both variants")
)

// SpecFromBlob returns the spec of a blob whose digest is known. Its digest
// code is the digest's own; a digest that is not a multihash has code 0.
func SpecFromBlob(b Blob) BlobSpec {
	s := BlobSpec{digest: b.Digest, size: b.Size, valid: true}
	if d, err := multihash.Decode(b.Digest); err == nil {
		s.code = d.Code
	}
	return s
}

// SpecFromDigestCode returns the spec of a blob of size bytes whose digest is
// to be computed with the multihash function code.
func SpecFromDigestCode(code, size uint64) BlobSpec {
	return BlobSpec{code: code, size: size, valid: true}
}

// Digest returns the blob's digest, when the spec names one.
func (s BlobSpec) Digest() (multihash.Multihash, bool) {
	return s.digest, len(s.digest) > 0
}

// DigestCode returns the code of the multihash function the blob's digest is
// computed with: the one the spec names, or its digest's own.
func (s BlobSpec) DigestCode() uint64 {
	return s.code
}

// Size returns the size of the blob.
func (s BlobSpec) Size() uint64 {
	return s.size
}

// model returns the spec's encoded form, which names the digest or the
// digest code but never both.
func (s BlobSpec) model() (BlobSpecModel, error) {
	if !s.valid {
		return BlobSpecModel{}, errNoVariant
	}
	if len(s.digest) > 0 {
		return BlobSpecModel{Digest: s.digest, Size: s.size}, nil
	}
	code := s.code
	return BlobSpecModel{DigestCode: &code, Size: s.size}, nil
}

func (s BlobSpec) MarshalCBOR(w io.Writer) error {
	m, err := s.model()
	if err != nil {
		return err
	}
	return m.MarshalCBOR(w)
}

func (s *BlobSpec) UnmarshalCBOR(r io.Reader) error {
	var m BlobSpecModel
	if err := m.UnmarshalCBOR(r); err != nil {
		return err
	}
	return s.fromModel(m)
}

func (s BlobSpec) MarshalDagJSON(w io.Writer) error {
	m, err := s.model()
	if err != nil {
		return err
	}
	return m.MarshalDagJSON(w)
}

func (s *BlobSpec) UnmarshalDagJSON(r io.Reader) error {
	var m BlobSpecModel
	if err := m.UnmarshalDagJSON(r); err != nil {
		return err
	}
	return s.fromModel(m)
}

func (s *BlobSpec) fromModel(m BlobSpecModel) error {
	*s = BlobSpec{}
	switch hasDigest := len(m.Digest) > 0; {
	case hasDigest && m.DigestCode == nil:
		d, err := multihash.Decode(m.Digest)
		if err != nil {
			return fmt.Errorf("decoding blob digest: %w", err)
		}
		*s = BlobSpec{digest: m.Digest, code: d.Code, size: m.Size, valid: true}
	case !hasDigest && m.DigestCode != nil:
		*s = SpecFromDigestCode(*m.DigestCode, m.Size)
	case hasDigest:
		return errBothVariants
	default:
		return errNoVariant
	}
	return nil
}

// RejectByDigest returns arguments rejecting space's allocation for the blob
// with digest.
func RejectByDigest(space did.DID, digest multihash.Multihash) RejectArguments {
	return RejectArguments{byDigest: &RejectDigestArguments{Space: space, Digest: digest}}
}

// RejectByAllocation returns arguments rejecting space's allocation made
// without a digest by the `/blob/allocate` task allocation.
func RejectByAllocation(space did.DID, allocation cid.Cid) RejectArguments {
	return RejectArguments{byAllocation: &RejectAllocationArguments{Space: space, Allocation: allocation}}
}

// Space returns the space whose allocation is rejected.
func (a RejectArguments) Space() did.DID {
	switch {
	case a.byDigest != nil:
		return a.byDigest.Space
	case a.byAllocation != nil:
		return a.byAllocation.Space
	}
	return did.DID{}
}

// Digest returns the digest of the rejected blob, when the arguments name one.
func (a RejectArguments) Digest() (multihash.Multihash, bool) {
	if a.byDigest == nil {
		return nil, false
	}
	return a.byDigest.Digest, true
}

// Allocation returns the `/blob/allocate` task of the rejected allocation,
// when the arguments name one instead of a digest.
func (a RejectArguments) Allocation() (cid.Cid, bool) {
	if a.byAllocation == nil {
		return cid.Undef, false
	}
	return a.byAllocation.Allocation, true
}

func (a RejectArguments) MarshalCBOR(w io.Writer) error {
	switch {
	case a.byDigest != nil && a.byAllocation == nil:
		return a.byDigest.MarshalCBOR(w)
	case a.byAllocation != nil && a.byDigest == nil:
		return a.byAllocation.MarshalCBOR(w)
	}
	return errNoVariant
}

func (a *RejectArguments) UnmarshalCBOR(r io.Reader) error {
	var m RejectArgumentsModel
	if err := m.UnmarshalCBOR(r); err != nil {
		return err
	}
	return a.fromModel(m)
}

func (a RejectArguments) MarshalDagJSON(w io.Writer) error {
	switch {
	case a.byDigest != nil && a.byAllocation == nil:
		return a.byDigest.MarshalDagJSON(w)
	case a.byAllocation != nil && a.byDigest == nil:
		return a.byAllocation.MarshalDagJSON(w)
	}
	return errNoVariant
}

func (a *RejectArguments) UnmarshalDagJSON(r io.Reader) error {
	var m RejectArgumentsModel
	if err := m.UnmarshalDagJSON(r); err != nil {
		return err
	}
	return a.fromModel(m)
}

func (a *RejectArguments) fromModel(m RejectArgumentsModel) error {
	*a = RejectArguments{}
	switch hasDigest := len(m.Digest) > 0; {
	case hasDigest && m.Allocation == nil:
		a.byDigest = &RejectDigestArguments{Space: m.Space, Digest: m.Digest}
	case !hasDigest && m.Allocation != nil:
		a.byAllocation = &RejectAllocationArguments{Space: m.Space, Allocation: *m.Allocation}
	case hasDigest:
		return errBothVariants
	default:
		return errNoVariant
	}
	return nil
}
