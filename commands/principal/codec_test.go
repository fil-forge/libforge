//go:build !codegen

package principal_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/fil-forge/libforge/commands/principal"
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/execution"
	"github.com/fil-forge/ucantone/testutil"
	"github.com/fil-forge/ucantone/ucan/invocation"
	"github.com/stretchr/testify/require"
)

var tenant = did.MustParse("did:plc:ewvi7nxzyoun6zhxrhs64oiz")

func TestInvalidateArgumentsRoundTrip(t *testing.T) {
	in := &principal.InvalidateArguments{
		Tenant:    tenant,
		Principal: "8f2c9e14",
	}

	var cb bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&cb))
	var outCBOR principal.InvalidateArguments
	require.NoError(t, outCBOR.UnmarshalCBOR(bytes.NewReader(cb.Bytes())))
	require.True(t, reflect.DeepEqual(*in, outCBOR), "CBOR round-trip mismatch:\n got %#v\nwant %#v", outCBOR, *in)

	var jb bytes.Buffer
	require.NoError(t, in.MarshalDagJSON(&jb))
	require.Equal(t, `{"principal":"8f2c9e14","tenant":"`+tenant.String()+`"}`, jb.String())
	var outJSON principal.InvalidateArguments
	require.NoError(t, outJSON.UnmarshalDagJSON(bytes.NewReader(jb.Bytes())), "json: %s", jb.String())
	require.True(t, reflect.DeepEqual(*in, outJSON), "DAG-JSON round-trip mismatch:\n got %#v\nwant %#v", outJSON, *in)
}

// The command is self-signed: Hilt is both issuer and subject, because no
// delegation names a principal and Swarf authorizes the invocation from its
// publisher list instead of a proof chain.
func TestInvalidateInvokeUnpack(t *testing.T) {
	hilt := testutil.RandomIssuer(t)
	swarf := testutil.RandomIssuer(t)

	args := &principal.InvalidateArguments{Tenant: tenant, Principal: "8f2c9e14"}
	inv, err := principal.Invalidate.Invoke(
		hilt,
		hilt.DID(),
		args,
		invocation.WithAudience(swarf.DID()),
		invocation.WithNoNonce(),
		invocation.WithNoExpiration(),
	)
	require.NoError(t, err)
	require.Equal(t, "/principal/invalidate", inv.Command().String())
	require.Equal(t, hilt.DID(), inv.Issuer())
	require.Equal(t, hilt.DID(), inv.Subject())
	require.Equal(t, swarf.DID(), inv.Audience())
	require.Empty(t, inv.Proofs())

	var seen *principal.InvalidateArguments
	handler := principal.Invalidate.Handler(func(req *binding.Request[*principal.InvalidateArguments], res *binding.Response[*principal.InvalidateOK]) error {
		seen = req.Task().Arguments()
		return res.SetSuccess(&principal.InvalidateOK{})
	})

	res, err := execution.NewResponse(inv.Task().Link(), execution.WithIssuer(swarf))
	require.NoError(t, err)
	require.NoError(t, handler(execution.NewRequest(t.Context(), inv), res))

	require.Equal(t, args, seen)

	out, err := principal.Invalidate.Unpack(res.Receipt())
	require.NoError(t, err)
	require.Equal(t, &principal.InvalidateOK{}, out)
}
