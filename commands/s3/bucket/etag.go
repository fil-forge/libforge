//go:build !codegen

// The functions below call generated encoders, which the generator builds
// the package without.

package bucket

import (
	"bytes"
	"io"

	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

// ETag returns the entity tag of d: the CID of its DAG-JSON encoding (CIDv1,
// dag-json, sha2-256), in double quotes as HTTP carries it. The encoding is
// canonical (see [StatementSet]), so two documents that differ only in
// statement order or duplicates share a tag. Hilt stores it with the policy
// and checks If-Match against it; Ingot answers GetBucketPolicy and
// PutBucketPolicy with it. Callers treat it as opaque.
func ETag(d PolicyDocument) string { return `"` + cidOf(&d).String() + `"` }

// StatementCID returns the CID of st's DAG-JSON encoding: its sets as maps
// with sorted keys, so nothing about how the caller built them reaches it.
func StatementCID(st Statement) cid.Cid { return cidOf(&st) }

// cidOf returns the CIDv1 (dag-json, sha2-256) of m's DAG-JSON encoding.
func cidOf(m interface{ MarshalDagJSON(io.Writer) error }) cid.Cid {
	var buf bytes.Buffer
	if err := m.MarshalDagJSON(&buf); err != nil {
		panic(err) // the sets are bounded by MaxSetSize, within the encoders' limits
	}
	sum, err := mh.Sum(buf.Bytes(), mh.SHA2_256, -1)
	if err != nil {
		panic(err) // sha2-256 with the default length cannot fail
	}
	return cid.NewCidV1(cid.DagJSON, sum)
}
