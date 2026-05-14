package didresolver_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fil-forge/libforge/didresolver"
	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/principal/ed25519/verifier"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/stretchr/testify/require"
)

func TestTieredResolver_ResolveDIDKey(t *testing.T) {
	didWeb, err := did.Parse("did:web:example.com")
	require.NoError(t, err)

	didKey, err := verifier.Parse("did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK")
	require.NoError(t, err)

	t.Run("returns from the first tier when it resolves", func(t *testing.T) {
		tier1 := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				return didKey, nil
			},
		}
		tier2 := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				t.Fatal("second tier should not be called when first tier resolves")
				return nil, nil
			},
		}

		resolver := &didresolver.TieredResolver{
			Tiers: []didresolver.DIDVerifierResolverFunc{tier1.ResolveDIDKey, tier2.ResolveDIDKey},
		}

		result, err := resolver.ResolveDIDKey(t.Context(), didWeb)
		require.NoError(t, err)
		require.Equal(t, didKey, result)
		require.Equal(t, 1, tier1.getCallCount())
		require.Equal(t, 0, tier2.getCallCount())
	})

	t.Run("falls through to later tier when earlier tiers fail", func(t *testing.T) {
		tier1 := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				return nil, fmt.Errorf("tier1 failed")
			},
		}
		tier2 := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				return nil, fmt.Errorf("tier2 failed")
			},
		}
		tier3 := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				return didKey, nil
			},
		}

		resolver := &didresolver.TieredResolver{
			Tiers: []didresolver.DIDVerifierResolverFunc{
				tier1.ResolveDIDKey,
				tier2.ResolveDIDKey,
				tier3.ResolveDIDKey,
			},
		}

		result, err := resolver.ResolveDIDKey(t.Context(), didWeb)
		require.NoError(t, err)
		require.Equal(t, didKey, result)
		require.Equal(t, 1, tier1.getCallCount())
		require.Equal(t, 1, tier2.getCallCount())
		require.Equal(t, 1, tier3.getCallCount())
	})

	t.Run("returns joined error when all tiers fail", func(t *testing.T) {
		tier1 := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				return nil, fmt.Errorf("tier1 specific error")
			},
		}
		tier2 := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				return nil, fmt.Errorf("tier2 specific error")
			},
		}

		resolver := &didresolver.TieredResolver{
			Tiers: []didresolver.DIDVerifierResolverFunc{tier1.ResolveDIDKey, tier2.ResolveDIDKey},
		}

		result, err := resolver.ResolveDIDKey(t.Context(), didWeb)
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "unable to resolve")
		require.Contains(t, err.Error(), "not resolvable by any tier")
		require.Contains(t, err.Error(), "tier1 specific error")
		require.Contains(t, err.Error(), "tier2 specific error")
		require.Equal(t, 1, tier1.getCallCount())
		require.Equal(t, 1, tier2.getCallCount())
	})

	t.Run("returns error with no tiers configured", func(t *testing.T) {
		resolver := &didresolver.TieredResolver{
			Tiers: []didresolver.DIDVerifierResolverFunc{},
		}

		result, err := resolver.ResolveDIDKey(t.Context(), didWeb)
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "unable to resolve")
		require.Contains(t, err.Error(), "not resolvable by any tier")
	})

	t.Run("works with a single tier", func(t *testing.T) {
		tier1 := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				return didKey, nil
			},
		}

		resolver := &didresolver.TieredResolver{
			Tiers: []didresolver.DIDVerifierResolverFunc{tier1.ResolveDIDKey},
		}

		result, err := resolver.ResolveDIDKey(t.Context(), didWeb)
		require.NoError(t, err)
		require.Equal(t, didKey, result)
		require.Equal(t, 1, tier1.getCallCount())
	})

	t.Run("composes with MapResolver tiers", func(t *testing.T) {
		didA, err := did.Parse("did:web:alice.example.com")
		require.NoError(t, err)
		didB, err := did.Parse("did:web:bob.example.com")
		require.NoError(t, err)
		didC, err := did.Parse("did:web:carol.example.com")
		require.NoError(t, err)

		keyA, err := verifier.Parse("did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK")
		require.NoError(t, err)
		keyB, err := verifier.Parse("did:key:z6Mkfriq1MqLBoPWecGoDLjguo1sB9brj6wT3qZ5BxkKpuP6")
		require.NoError(t, err)

		mapA, err := didresolver.NewMapResolver(map[string]string{didA.String(): keyA.DID().String()})
		require.NoError(t, err)
		mapB, err := didresolver.NewMapResolver(map[string]string{didB.String(): keyB.DID().String()})
		require.NoError(t, err)

		resolver := &didresolver.TieredResolver{
			Tiers: []didresolver.DIDVerifierResolverFunc{mapA.Resolve, mapB.Resolve},
		}

		// Resolves via the first tier
		resA, err := resolver.ResolveDIDKey(t.Context(), didA)
		require.NoError(t, err)
		require.Equal(t, keyA, resA)

		// Falls through to the second tier
		resB, err := resolver.ResolveDIDKey(t.Context(), didB)
		require.NoError(t, err)
		require.Equal(t, keyB, resB)

		// Not in any tier
		_, err = resolver.ResolveDIDKey(t.Context(), didC)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not resolvable by any tier")
	})

	t.Run("propagates context to tiers", func(t *testing.T) {
		type ctxKey string
		key := ctxKey("marker")
		ctx := context.WithValue(t.Context(), key, "value")

		var seen string
		tier1 := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				if v, ok := ctx.Value(key).(string); ok {
					seen = v
				}
				return didKey, nil
			},
		}

		resolver := &didresolver.TieredResolver{
			Tiers: []didresolver.DIDVerifierResolverFunc{tier1.ResolveDIDKey},
		}

		_, err := resolver.ResolveDIDKey(ctx, didWeb)
		require.NoError(t, err)
		require.Equal(t, "value", seen)
	})
}
