package attestation_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"

	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/did/key"
	"github.com/fil-forge/ucantone/ucan/command"
	"github.com/fil-forge/ucantone/ucan/delegation"
	"github.com/fil-forge/ucantone/ucan/invocation"
	"github.com/fil-forge/ucantone/validator"

	"github.com/fil-forge/libforge/attestation"
	"github.com/fil-forge/libforge/attestation/didmailto"
	"github.com/fil-forge/libforge/commands/ucan/attest"
	"github.com/fil-forge/libforge/testutil"
)

// TestChainedAttestation verifies that attested signatures chain correctly when
// the authority is itself a did:mailto attested by a root key. In other words,
// this demonstrates that the signature is validated as an invocation using the
// same DID resolvers and verifier factories that the top `ValidateToken` got.
func TestChainedAttestation(t *testing.T) {
	// root signs with real ed25519 keys (did:key)
	root := testutil.RandomIssuer(t)

	// service is a did:mailto DID that signs via attestation from root
	serviceDID, err := did.Parse("did:mailto:service.example.com:svc")
	require.NoError(t, err)
	service := attestation.Attest(t.Context(), serviceDID, root)

	// alice is a did:mailto DID that signs via attestation from service
	aliceDID, err := did.Parse("did:mailto:example.com:alice")
	require.NoError(t, err)
	alice := attestation.Attest(t.Context(), aliceDID, service)

	del, err := delegation.Delegate(
		alice,
		testutil.RandomDID(t),
		alice.DID(),
		command.MustParse("/example/command"),
	)
	require.NoError(t, err)

	encoded, err := delegation.Encode(del)
	require.NoError(t, err)

	decoded, err := delegation.Decode(encoded)
	require.NoError(t, err)

	// The resolver needs to know both mailto DIDs with their distinct authorities.
	resolver := did.ResolverMap{
		"key": key.Resolver,
		"mailto": did.ResolverFunc(func(ctx context.Context, d did.DID) (did.Document, error) {
			authority := serviceDID // default: alice and others attested by service
			if d == serviceDID {
				authority = root.DID() // service itself is attested by root
			}
			return didmailto.NewResolver(authority).Resolve(ctx, d)
		}),
	}
	factories := validator.DefaultFactories()
	factories[attestation.Type] = attestation.NewVerifierFactory(resolver, factories)
	err = validator.ValidateToken(t.Context(), decoded,
		validator.WithDIDResolver(resolver),
		validator.WithVerifierFactories(factories),
	)
	require.NoError(t, err)
}

func TestSigner(t *testing.T) {
	authority := testutil.RandomIssuer(t)
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

	t.Run("signs data correctly", func(t *testing.T) {
		require.Equal(t, del.Signature().Header().SignatureAlgorithm(), attestation.Algorithm{})
		sigBytes := del.Signature().Bytes()
		require.NotEmpty(t, sigBytes)

		inv, err := invocation.Decode(sigBytes)
		require.NoError(t, err)

		require.Equal(t, authority.DID(), inv.Issuer())
		require.Equal(t, did.Undef, inv.Audience())
		require.Equal(t, authority.DID(), inv.Subject())
		require.Equal(t, attest.Proof.Command, inv.Command())

		msgDigest, err := mh.Sum(del.SignedBytes(), mh.SHA2_256, -1)
		require.NoError(t, err)
		var proofArgs attest.ProofArguments
		err = proofArgs.UnmarshalCBOR(bytes.NewReader(inv.ArgumentsBytes()))
		require.NoError(t, err)
		require.Equal(t, attest.ProofArguments{Proof: cid.NewCidV1(cid.Raw, msgDigest)}, proofArgs)
	})

	t.Run("delegation round-trips through CBOR and verifies", func(t *testing.T) {
		encoded, err := delegation.Encode(del)
		require.NoError(t, err)

		decoded, err := delegation.Decode(encoded)
		require.NoError(t, err)

		resolver := did.ResolverMap{
			"key":    key.Resolver,
			"mailto": didmailto.NewResolver(authority.DID()),
		}
		factories := validator.DefaultFactories()
		factories[attestation.Type] = attestation.NewVerifierFactory(resolver, factories)
		err = validator.ValidateToken(t.Context(), decoded,
			validator.WithDIDResolver(resolver),
			validator.WithVerifierFactories(factories),
		)
		require.NoError(t, err)
	})
}
