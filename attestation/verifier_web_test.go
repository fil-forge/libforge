package attestation_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/did/key"
	"github.com/fil-forge/ucantone/ucan/command"
	"github.com/fil-forge/ucantone/ucan/delegation"
	"github.com/fil-forge/ucantone/validator"

	"github.com/fil-forge/libforge/attestation"
	"github.com/fil-forge/libforge/attestation/didmailto"
	"github.com/fil-forge/libforge/identity"
	"github.com/fil-forge/libforge/testutil"
)

// TestSigner_WebAuthority exercises an attestation whose authority is a did:web
// service (e.g. did:web:upload), resolved via its DID document — the real
// service flow. The existing TestSigner uses a did:key authority, which the
// default key.Resolver handles, so it never caught that the attestation verifier
// re-resolved the authority with the did:key-only default resolver. With a
// did:web authority that path fails ("signature mismatch"); this test guards the
// fix that verifies with the already-resolved authority verifier.
func TestSigner_WebAuthority(t *testing.T) {
	authority, err := identity.New("", "did:web:example.com")
	require.NoError(t, err)

	doc, err := authority.DIDDocument()
	require.NoError(t, err)

	alice, err := did.Parse("did:mailto:example.com:alice")
	require.NoError(t, err)

	issuer := attestation.Attest(t.Context(), alice, authority)

	del, err := delegation.Delegate(
		issuer,
		testutil.RandomDID(t),
		issuer.DID(),
		command.MustParse("/example/command"),
	)
	require.NoError(t, err)

	encoded, err := delegation.Encode(del)
	require.NoError(t, err)
	decoded, err := delegation.Decode(encoded)
	require.NoError(t, err)

	// Serve the authority's generated did:web document.
	webResolver := did.ResolverFunc(func(_ context.Context, d did.DID) (did.Document, error) {
		if d == authority.DID() {
			return doc, nil
		}
		return did.Document{}, fmt.Errorf("unexpected did %s", d)
	})
	resolver := did.ResolverMap{
		"key":    key.Resolver,
		"web":    webResolver,
		"mailto": didmailto.NewResolver(authority.DID()),
	}
	factories := validator.DefaultFactories()
	factories[attestation.Type] = attestation.NewVerifierFactory(resolver, factories)

	err = validator.ValidateToken(t.Context(), decoded,
		validator.WithDIDResolver(resolver),
		validator.WithVerifierFactories(factories),
	)
	require.NoError(t, err)
}
