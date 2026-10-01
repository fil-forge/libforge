//go:build !codegen

package blob

import (
	"fmt"
	"io"

	"github.com/fil-forge/ucantone/errors"
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

// SpecFromDigest returns the spec of a blob of size bytes whose digest is
// known. Its digest code is the digest's own; a digest that is not a multihash
// has code 0.
func SpecFromDigest(digest multihash.Multihash, size uint64) BlobSpec {
	s := BlobSpec{digest: digest, size: size, valid: true}
	if d, err := multihash.Decode(digest); err == nil {
		s.code = d.Code
	}
	return s
}

// SpecFromBlob returns the spec of a blob whose digest is known.
func SpecFromBlob(b Blob) BlobSpec {
	return SpecFromDigest(b.Digest, b.Size)
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
