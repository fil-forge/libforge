//go:build !codegen

package http_test

import (
	"bytes"
	"testing"

	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/libforge/commands/http"
	"github.com/fil-forge/libforge/testutil"
	"github.com/stretchr/testify/require"
)

func TestPutOKRoundTrip(t *testing.T) {
	in := http.PutOK{Blob: &http.PutBlob{Digest: testutil.RandomMultihash(t)}}
	var buf bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&buf))
	var out http.PutOK
	require.NoError(t, out.UnmarshalCBOR(&buf))
	require.Equal(t, in, out)

	var js bytes.Buffer
	require.NoError(t, in.MarshalDagJSON(&js))
	var outJS http.PutOK
	require.NoError(t, outJS.UnmarshalDagJSON(&js))
	require.Equal(t, in, outJS)
}

// An empty result encodes as the empty map it was before it could carry a
// digest, so existing receipts decode and encode unchanged.
func TestPutOKEmptyEncodesAsUnit(t *testing.T) {
	var want, got bytes.Buffer
	require.NoError(t, (&commands.Unit{}).MarshalCBOR(&want))
	require.NoError(t, (&http.PutOK{}).MarshalCBOR(&got))
	require.Equal(t, want.Bytes(), got.Bytes())

	var out http.PutOK
	require.NoError(t, out.UnmarshalCBOR(bytes.NewReader(want.Bytes())))
	require.Nil(t, out.Blob)
}
