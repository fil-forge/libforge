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

}

func TestBlobSpecAccessors(t *testing.T) {
	digest := testutil.RandomMultihash(t)
	byDigest := blob.SpecFromBlob(blob.Blob{Digest: digest, Size: 7})
	got, ok := byDigest.Digest()
	require.True(t, ok)
	require.Equal(t, digest, got)
	require.Equal(t, uint64(multicodec.Sha2_256), byDigest.DigestCode(), "a digest has its own code")
	require.EqualValues(t, 7, byDigest.Size())

	byCode := blob.SpecFromDigestCode(uint64(multicodec.Sha2_256), 9)
	_, ok = byCode.Digest()
	require.False(t, ok)
	require.Equal(t, uint64(multicodec.Sha2_256), byCode.DigestCode())
	require.EqualValues(t, 9, byCode.Size())
}

// A spec naming a digest decodes only when the digest is a multihash, since
// its digest code comes from it.
func TestBlobSpecRefusesMalformedDigest(t *testing.T) {
	m := blob.BlobSpecModel{Digest: []byte{0xff}, Size: 1}
	var buf bytes.Buffer
	require.NoError(t, m.MarshalCBOR(&buf))
	var out blob.BlobSpec
	require.Error(t, out.UnmarshalCBOR(&buf))
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

// A spec decodes through its model, which holds the fields of both variants:
// exactly one variant's field must be set.
func TestBlobSpecModelRejectsInvalidCBOR(t *testing.T) {
	digest := testutil.RandomMultihash(t)
	code := uint64(multicodec.Sha2_256)

	for name, m := range map[string]blob.BlobSpecModel{
		"neither": {Size: 1},
		"both":    {Digest: digest, DigestCode: &code, Size: 1},
	} {
		t.Run("spec "+name, func(t *testing.T) {
			var buf bytes.Buffer
			require.NoError(t, m.MarshalCBOR(&buf))
			var out blob.BlobSpec
			require.Error(t, out.UnmarshalCBOR(&buf))
		})
	}
}

// The model's generated decoder bounds each field as it reads it, so a spec
// decodes in one pass with no intermediate copy of its fields.
func TestBlobSpecRefusesOversizedDigest(t *testing.T) {
	// {"digest": <3 MiB of bytes>, "size": 1}, written by hand: the generated
	// encoder refuses a digest this large too.
	n := 3 << 20
	var buf bytes.Buffer
	buf.WriteByte(0xa2)
	buf.WriteString("\x66digest")
	buf.Write([]byte{0x5a, byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)})
	buf.Write(bytes.Repeat([]byte{1}, n))
	buf.WriteString("\x64size\x01")
	var out blob.BlobSpec
	require.ErrorContains(t, out.UnmarshalCBOR(&buf), "t.Digest: byte array too large")
}
