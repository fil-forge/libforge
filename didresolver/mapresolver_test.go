package didresolver_test

import (
	"testing"

	"github.com/fil-forge/libforge/didresolver"
	"github.com/fil-forge/ucantone/did"
	"github.com/stretchr/testify/require"
)

func TestPrincipalResolver(t *testing.T) {
	p0, err := did.Parse("did:web:example.com")
	require.NoError(t, err)
	r, err := did.Parse("did:key:z6MkghfetkhrBZwUupJrv8MmYDH1JhKCQCGj1trbaZPA3dAd")
	require.NoError(t, err)
	p1, err := did.Parse("did:web:example.org")
	require.NoError(t, err)

	pm := map[string]string{p0.String(): r.String()}
	ppr, err := didresolver.NewMapResolver(pm)
	require.NoError(t, err)

	resolved, err := ppr.Resolve(t.Context(), p0)
	require.NoError(t, err)
	// Resolver wraps the underlying did:key verifier so it announces the
	// requested did:web — required for ucantone token.VerifySignature, which
	// compares issuer DID against verifier DID before checking signature bytes.
	require.Equal(t, p0, resolved.DID())

	// cannot resolve DID not in mapping
	_, err = ppr.Resolve(t.Context(), p1)
	require.ErrorContains(t, err, "not found in mapping")
}
