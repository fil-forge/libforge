//go:build !codegen

// The functions below call generated encoders, which the generator builds
// the package without.

package bucket

import (
	"bytes"
	"slices"
	"strings"

	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

// Normalize returns a copy of d in canonical form, its Version kept: the
// statements deduplicated and ordered by their CID (see [StatementCID]). The
// principal, action and resource sets need nothing, being sets. Two policies
// that differ only in statement order or in duplicate statements normalize
// to the same value.
func Normalize(d BucketPolicy) BucketPolicy {
	type keyed struct {
		key string
		st  Statement
	}
	keyedStatements := make([]keyed, 0, len(d.Statements))
	for _, st := range d.Statements {
		keyedStatements = append(keyedStatements, keyed{key: StatementCID(st).String(), st: st})
	}
	slices.SortFunc(keyedStatements, func(a, b keyed) int { return strings.Compare(a.key, b.key) })
	keyedStatements = slices.CompactFunc(keyedStatements, func(a, b keyed) bool { return a.key == b.key })
	statements := make([]Statement, len(keyedStatements))
	for i, k := range keyedStatements {
		statements[i] = k.st
	}
	return BucketPolicy{Version: d.Version, Statements: statements}
}

// ETag returns the entity tag of d: the CID of the DAG-JSON encoding of the
// [Normalize]d document (CIDv1, dag-json, sha2-256), its statements in CID
// order and each statement's sets as maps, in double quotes as HTTP carries
// it. Hilt stores it with the policy and checks If-Match against it; Ingot
// answers GetBucketPolicy and PutBucketPolicy with it. Callers treat it as
// opaque.
func ETag(d BucketPolicy) string {
	n := Normalize(d)
	var buf bytes.Buffer
	if err := n.MarshalDagJSON(&buf); err != nil {
		panic(err) // the sets are bounded by MaxSetSize, within the encoder's map limit
	}
	return `"` + dagJSONCID(buf.Bytes()).String() + `"`
}

// StatementCID returns the CID of st's DAG-JSON encoding: its sets as maps
// with sorted keys, so nothing about how the caller built them reaches it.
func StatementCID(st Statement) cid.Cid {
	var buf bytes.Buffer
	if err := st.MarshalDagJSON(&buf); err != nil {
		panic(err) // see ETag
	}
	return dagJSONCID(buf.Bytes())
}

// dagJSONCID returns the CIDv1 (dag-json, sha2-256) of raw.
func dagJSONCID(raw []byte) cid.Cid {
	sum, err := mh.Sum(raw, mh.SHA2_256, -1)
	if err != nil {
		panic(err) // sha2-256 with the default length cannot fail
	}
	return cid.NewCidV1(cid.DagJSON, sum)
}
