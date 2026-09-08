//go:build !codegen

package routing_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/fil-forge/libforge/commands/routing"
	"github.com/fil-forge/ucantone/did"
	"github.com/stretchr/testify/require"
)

var (
	nodeA = did.MustParse("did:key:z6MkjFRxLLGdBqQSLkZbVnuwUFiomK8eGBkPtim9ETvP7vec")
	nodeB = did.MustParse("did:key:z6MkmNBgCewjYfEDTdKLpHkbMWUogJk29CxmiVdLeW4Kz3UG")
)

type wire interface {
	MarshalCBOR(w io.Writer) error
	UnmarshalCBOR(r io.Reader) error
	MarshalDagJSON(w io.Writer) error
	UnmarshalDagJSON(r io.Reader) error
}

// roundTrip encodes in as CBOR and DAG-JSON, decodes each and asserts equality.
// It returns the DAG-JSON encoding.
func roundTrip[T any, PT interface {
	*T
	wire
}](t *testing.T, in PT) string {
	t.Helper()
	var cb bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&cb))
	outCBOR := PT(new(T))
	require.NoError(t, outCBOR.UnmarshalCBOR(bytes.NewReader(cb.Bytes())))
	require.True(t, reflect.DeepEqual(*in, *outCBOR), "CBOR round-trip mismatch:\n got %#v\nwant %#v", *outCBOR, *in)

	var jb bytes.Buffer
	require.NoError(t, in.MarshalDagJSON(&jb))
	outJSON := PT(new(T))
	require.NoError(t, outJSON.UnmarshalDagJSON(bytes.NewReader(jb.Bytes())), "json: %s", jb.String())
	require.True(t, reflect.DeepEqual(*in, *outJSON), "DAG-JSON round-trip mismatch:\n got %#v\nwant %#v", *outJSON, *in)
	return jb.String()
}

func TestPutArgumentsRoundTrip(t *testing.T) {
	in := &routing.PutArguments{Candidates: routing.CandidateSet{Entries: map[did.DID]routing.Candidate{
		nodeB: {},
		nodeA: {},
	}}}
	// Keys are sorted and each candidate is an empty object.
	require.Equal(t, `{"candidates":{"`+nodeA.String()+`":{},"`+nodeB.String()+`":{}}}`, roundTrip(t, in))
}

func TestCandidateSetEmpty(t *testing.T) {
	in := &routing.CandidateSet{Entries: map[did.DID]routing.Candidate{}}
	require.Equal(t, `{}`, roundTrip(t, in))
}

func TestCandidateSetRejectsInvalidDID(t *testing.T) {
	var out routing.CandidateSet
	require.Error(t, out.UnmarshalDagJSON(bytes.NewReader([]byte(`{"not-a-did":{}}`))))
}

func TestUseArgumentsRoundTrip(t *testing.T) {
	policy := nodeA
	require.Equal(t, `{"policy":"`+policy.String()+`"}`, roundTrip(t, &routing.UseArguments{Policy: &policy}))
	// Absent policy omits the key: this is the "clear" form.
	require.Equal(t, `{}`, roundTrip(t, &routing.UseArguments{}))
}
