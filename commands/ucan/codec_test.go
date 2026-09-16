//go:build !codegen

package ucan_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	ucancmds "github.com/fil-forge/libforge/commands/ucan"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
)

var (
	rcptA = cid.MustParse("bafkreiaixnpf23vkyecj5xqispjq5ubcwgsntnnurw2bjaz5pmjhndcxqi")
	rcptB = cid.MustParse("bafkreibme22gw2h7y2h7tg2fhqotaqjucnbc24deqo72b6mkl2egezxhvy")
)

type wire interface {
	MarshalCBOR(w io.Writer) error
	UnmarshalCBOR(r io.Reader) error
	MarshalDagJSON(w io.Writer) error
	UnmarshalDagJSON(r io.Reader) error
}

// roundTrip encodes in as CBOR and DAG-JSON, decodes each and asserts
// equality. It returns the DAG-JSON encoding.
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

// TestConcludeArgumentsOne pins the one-receipt case: still a list, so every
// delivery has the same shape on the wire.
func TestConcludeArgumentsOne(t *testing.T) {
	in := &ucancmds.ConcludeArguments{Receipts: []cid.Cid{rcptA}}
	require.Equal(t, `{"receipts":[{"/":"`+rcptA.String()+`"}]}`, roundTrip(t, in))
}

func TestConcludeArgumentsMany(t *testing.T) {
	in := &ucancmds.ConcludeArguments{Receipts: []cid.Cid{rcptA, rcptB}}
	require.Equal(t,
		`{"receipts":[{"/":"`+rcptA.String()+`"},{"/":"`+rcptB.String()+`"}]}`,
		roundTrip(t, in))
}

// Delivering nothing is a no-op that encodes as an empty list. An empty slice
// decodes back as nil, which every reader treats the same way.
func TestConcludeArgumentsEmpty(t *testing.T) {
	var cb bytes.Buffer
	require.NoError(t, (&ucancmds.ConcludeArguments{}).MarshalCBOR(&cb))

	var out ucancmds.ConcludeArguments
	require.NoError(t, out.UnmarshalCBOR(bytes.NewReader(cb.Bytes())))
	require.Empty(t, out.Receipts)

	var jb bytes.Buffer
	require.NoError(t, (&ucancmds.ConcludeArguments{}).MarshalDagJSON(&jb))

	var outJSON ucancmds.ConcludeArguments
	require.NoError(t, outJSON.UnmarshalDagJSON(bytes.NewReader(jb.Bytes())))
	require.Empty(t, outJSON.Receipts)
}
