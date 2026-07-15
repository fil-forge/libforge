//go:build !codegen

package blob_test

import (
	"bytes"
	"testing"

	"github.com/fil-forge/libforge/commands/blob"
	"github.com/fil-forge/libforge/testutil"
	"github.com/stretchr/testify/require"
)

// Round-trips UnallocateArguments through cbor with and without the
// optional Cause.
func TestUnallocateArgumentsRoundTrip(t *testing.T) {
	t.Run("with cause", func(t *testing.T) {
		cause := testutil.RandomCID(t)
		in := blob.UnallocateArguments{
			Space:  testutil.RandomDID(t),
			Digest: testutil.RandomMultihash(t),
			Cause:  &cause,
		}
		var buf bytes.Buffer
		require.NoError(t, in.MarshalCBOR(&buf))
		var out blob.UnallocateArguments
		require.NoError(t, out.UnmarshalCBOR(&buf))
		require.Equal(t, in.Space, out.Space)
		require.Equal(t, in.Digest, out.Digest)
		require.NotNil(t, out.Cause)
		require.Equal(t, cause, *out.Cause)
	})

	t.Run("without cause", func(t *testing.T) {
		in := blob.UnallocateArguments{
			Space:  testutil.RandomDID(t),
			Digest: testutil.RandomMultihash(t),
		}
		var buf bytes.Buffer
		require.NoError(t, in.MarshalCBOR(&buf))
		var out blob.UnallocateArguments
		require.NoError(t, out.UnmarshalCBOR(&buf))
		require.Equal(t, in.Space, out.Space)
		require.Equal(t, in.Digest, out.Digest)
		require.Nil(t, out.Cause)
	})
}
