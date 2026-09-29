//go:build !codegen

package blob_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fil-forge/libforge/commands/blob"
	"github.com/fil-forge/libforge/testutil"
	"github.com/multiformats/go-multicodec"
	"github.com/stretchr/testify/require"
)

// A spec with a digest encodes exactly as a Blob does, so invocations that
// name a digest keep their task links.
func TestBlobSpecWithDigestEncodesAsBlob(t *testing.T) {
	b := blob.Blob{Digest: testutil.RandomMultihash(t), Size: 1024}
	spec := blob.SpecFromBlob(b)

	var wantCBOR, gotCBOR bytes.Buffer
	require.NoError(t, b.MarshalCBOR(&wantCBOR))
	require.NoError(t, spec.MarshalCBOR(&gotCBOR))
	require.Equal(t, wantCBOR.Bytes(), gotCBOR.Bytes())

	var wantJSON, gotJSON bytes.Buffer
	require.NoError(t, b.MarshalDagJSON(&wantJSON))
	require.NoError(t, spec.MarshalDagJSON(&gotJSON))
	require.Equal(t, wantJSON.String(), gotJSON.String())
}

func TestBlobSpecRoundTrip(t *testing.T) {
	b := blob.Blob{Digest: testutil.RandomMultihash(t), Size: 7}
	code := uint64(multicodec.Sha2_256)
	for name, in := range map[string]blob.BlobSpec{
		"digest":      blob.SpecFromBlob(b),
		"digest code": blob.SpecFromDigestCode(code, 2097152),
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			require.NoError(t, in.MarshalCBOR(&buf))
			var out blob.BlobSpec
			require.NoError(t, out.UnmarshalCBOR(&buf))
			require.Equal(t, in, out)

			var js bytes.Buffer
			require.NoError(t, in.MarshalDagJSON(&js))
			var outJS blob.BlobSpec
			require.NoError(t, outJS.UnmarshalDagJSON(&js))
			require.Equal(t, in, outJS)
		})
	}

	gotBlob, ok := blob.SpecFromBlob(b).Blob()
	require.True(t, ok)
	require.Equal(t, b, gotBlob)
	gotCode, ok := blob.SpecFromDigestCode(code, 9).DigestCode()
	require.True(t, ok)
	require.Equal(t, blob.BlobDigestCode{DigestCode: code, Size: 9}, gotCode)
	require.EqualValues(t, 9, blob.SpecFromDigestCode(code, 9).Size())
}

// A spec travels inside the arguments' generated codecs: the nested union
// round-trips in both encodings.
func TestAddArgumentsWithDigestCodeRoundTrip(t *testing.T) {
	in := blob.AddArguments{Blob: blob.SpecFromDigestCode(uint64(multicodec.Sha2_256), 42)}
	var buf bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&buf))
	var out blob.AddArguments
	require.NoError(t, out.UnmarshalCBOR(&buf))
	require.Equal(t, in, out)

	var js bytes.Buffer
	require.NoError(t, in.MarshalDagJSON(&js))
	require.Equal(t, `{"blob":{"digestCode":18,"size":42}}`, js.String())
	var outJS blob.AddArguments
	require.NoError(t, outJS.UnmarshalDagJSON(&js))
	require.Equal(t, in, outJS)
}

// A spec holding neither variant cannot be encoded, and an encoding holding
// neither or both cannot be decoded.
func TestBlobSpecRejectsInvalidUnion(t *testing.T) {
	var buf bytes.Buffer
	require.Error(t, blob.BlobSpec{}.MarshalCBOR(&buf))
	require.Error(t, blob.BlobSpec{}.MarshalDagJSON(&buf))

	for name, js := range map[string]string{
		"neither": `{"size":1}`,
		"both":    `{"digest":{"/":{"bytes":"EiA"}},"digestCode":18,"size":1}`,
	} {
		t.Run(name, func(t *testing.T) {
			var out blob.BlobSpec
			require.Error(t, out.UnmarshalDagJSON(strings.NewReader(js)))
		})
	}
}

func TestRejectArgumentsByAllocation(t *testing.T) {
	space, alloc := testutil.RandomDID(t), testutil.RandomCID(t)
	in := blob.RejectByAllocation(space, alloc)

	var buf bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&buf))
	var out blob.RejectArguments
	require.NoError(t, out.UnmarshalCBOR(&buf))
	require.Equal(t, space, out.Space())
	_, hasDigest := out.Digest()
	require.False(t, hasDigest)
	got, ok := out.Allocation()
	require.True(t, ok)
	require.Equal(t, alloc, got)

	var js bytes.Buffer
	require.NoError(t, in.MarshalDagJSON(&js))
	var outJS blob.RejectArguments
	require.NoError(t, outJS.UnmarshalDagJSON(&js))
	require.Equal(t, in, outJS)

	require.Error(t, blob.RejectArguments{}.MarshalCBOR(&buf))
	var bad blob.RejectArguments
	require.Error(t, bad.UnmarshalDagJSON(strings.NewReader(`{"space":"did:key:z6Mk"}`)))
}

// An abort names only the cause when the add named only a digest code.
func TestAbortArgumentsWithoutDigest(t *testing.T) {
	in := blob.AbortArguments{Cause: testutil.RandomCID(t)}
	var buf bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&buf))
	var out blob.AbortArguments
	require.NoError(t, out.UnmarshalCBOR(&buf))
	require.Empty(t, out.Digest)
	require.Equal(t, in.Cause, out.Cause)
}
