package egress_test

import (
	"bytes"
	"testing"

	"github.com/fil-forge/libforge/commands/egress"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
)

func TestTrackArgumentsBatchURL(t *testing.T) {
	batch, err := cid.Parse("bafkreigh2akiscaildcqabsyg3dfr6chu3fgpregiymsck7e7aqa4s52zy")
	require.NoError(t, err)
	const endpoint = "https://gateway.example.com/.well-known/egress/{cid}?x=1"

	// The template is sent as written: the placeholder is not escaped.
	var buf bytes.Buffer
	require.NoError(t, (&egress.TrackArguments{Receipts: batch, Endpoint: endpoint}).MarshalCBOR(&buf))
	require.Contains(t, buf.String(), endpoint)
	var args egress.TrackArguments
	require.NoError(t, args.UnmarshalCBOR(&buf))
	require.Equal(t, endpoint, args.Endpoint)

	u, err := args.BatchURL()
	require.NoError(t, err)
	require.Equal(t, "https://gateway.example.com/.well-known/egress/"+batch.String()+"?x=1", u.String())
}

func TestResolveEndpointRefusesABadTemplate(t *testing.T) {
	batch, err := cid.Parse("bafkreigh2akiscaildcqabsyg3dfr6chu3fgpregiymsck7e7aqa4s52zy")
	require.NoError(t, err)
	for _, endpoint := range []string{
		"",
		"https://gateway.example.com/egress/",
		"/egress/{cid}",
		"file:///egress/{cid}",
		"https://gateway.example.com/%zz/{cid}",
	} {
		_, err := egress.ResolveEndpoint(endpoint, batch)
		require.Error(t, err, endpoint)
	}

	_, err = egress.TrackArguments{Endpoint: "https://gateway.example.com/{cid}"}.BatchURL()
	require.Error(t, err)
}
