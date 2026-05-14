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
	require.Equal(t, r, resolved.DID())

	// cannot resolve DID not in mapping
	_, err = ppr.Resolve(t.Context(), p1)
	require.ErrorContains(t, err, "not found in mapping")
}
