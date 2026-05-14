package didresolver_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fil-forge/libforge/didresolver"
	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/principal/ed25519/verifier"
	"github.com/fil-forge/ucantone/ucan"
	verrs "github.com/fil-forge/ucantone/validator/errors"
	"github.com/stretchr/testify/require"
)

type mockResolver struct {
	resolveFn didresolver.DIDVerifierResolverFunc
	callCount int32
}

func (m *mockResolver) ResolveDIDKey(ctx context.Context, input did.DID) (ucan.Verifier, error) {
	atomic.AddInt32(&m.callCount, 1)
	if m.resolveFn != nil {
		return m.resolveFn(ctx, input)
	}
	return nil, fmt.Errorf("mock error")
}

func (m *mockResolver) getCallCount() int {
	return int(atomic.LoadInt32(&m.callCount))
}

func TestNewCachedResolver(t *testing.T) {
	t.Run("creates resolver with valid TTL", func(t *testing.T) {
		mockWrapped := &mockResolver{}
		resolver, err := didresolver.NewCachedResolver(mockWrapped.ResolveDIDKey, 5*time.Minute)
		require.NoError(t, err)
		require.NotNil(t, resolver)
	})

	t.Run("creates resolver with zero TTL", func(t *testing.T) {
		mockWrapped := &mockResolver{}
		resolver, err := didresolver.NewCachedResolver(mockWrapped.ResolveDIDKey, 0)
		require.NoError(t, err)
		require.NotNil(t, resolver)
	})
}

func TestCachedResolver_ResolveDIDKey(t *testing.T) {
	t.Run("caches successful resolution", func(t *testing.T) {
		didWeb, err := did.Parse("did:web:example.com")
		require.NoError(t, err)

		didKey, err := verifier.Parse("did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK")
		require.NoError(t, err)

		mock := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				if input.String() == didWeb.String() {
					return didKey, nil
				}
				return nil, fmt.Errorf("not found")
			},
		}

		resolver, err := didresolver.NewCachedResolver(mock.ResolveDIDKey, 100*time.Millisecond)
		require.NoError(t, err)

		// First call should hit the wrapped resolver
		result1, err1 := resolver.Resolve(t.Context(), didWeb)
		require.Nil(t, err1)
		require.Equal(t, didKey, result1)
		require.Equal(t, 1, mock.getCallCount())

		// Second call should use cache
		result2, err2 := resolver.Resolve(t.Context(), didWeb)
		require.Nil(t, err2)
		require.Equal(t, didKey, result2)
		require.Equal(t, 1, mock.getCallCount()) // No additional call

		// Wait for cache to expire
		time.Sleep(150 * time.Millisecond)

		// Third call should hit the wrapped resolver again
		result3, err3 := resolver.Resolve(t.Context(), didWeb)
		require.Nil(t, err3)
		require.Equal(t, didKey, result3)
		require.Equal(t, 2, mock.getCallCount())
	})

	t.Run("does not cache errors", func(t *testing.T) {
		didWeb, err := did.Parse("did:web:example.com")
		require.NoError(t, err)

		mock := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				return nil, verrs.NewDIDKeyResolutionError(input, fmt.Errorf("resolution failed"))
			},
		}

		resolver, err := didresolver.NewCachedResolver(mock.ResolveDIDKey, 100*time.Millisecond)
		require.NoError(t, err)

		// First call
		result1, err1 := resolver.Resolve(t.Context(), didWeb)
		require.NotNil(t, err1)
		require.Nil(t, result1)
		require.Equal(t, 1, mock.getCallCount())

		// Second call should still hit the wrapped resolver (errors not cached)
		result2, err2 := resolver.Resolve(t.Context(), didWeb)
		require.NotNil(t, err2)
		require.Nil(t, result2)
		require.Equal(t, 2, mock.getCallCount())
	})

	t.Run("handles concurrent access", func(t *testing.T) {
		didWeb, err := did.Parse("did:web:example.com")
		require.NoError(t, err)

		didKey, err := verifier.Parse("did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK")
		require.NoError(t, err)

		var resolverCalls int32
		mock := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				atomic.AddInt32(&resolverCalls, 1)
				time.Sleep(10 * time.Millisecond) // Simulate slow resolution
				return didKey, nil
			},
		}

		resolver, err := didresolver.NewCachedResolver(mock.ResolveDIDKey, 1*time.Second)
		require.NoError(t, err)

		var wg sync.WaitGroup
		results := make([]ucan.Verifier, 10)
		errors := make([]error, 10)

		// Launch 10 concurrent requests
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx], errors[idx] = resolver.Resolve(t.Context(), didWeb)
			}(i)
		}

		wg.Wait()

		// All should succeed with the same result
		for i := 0; i < 10; i++ {
			require.Nil(t, errors[i])
			require.Equal(t, didKey, results[i])
		}

		// Due to caching, we expect fewer calls than requests
		// But with very fast concurrent access, all 10 might hit before the first one finishes
		actualCalls := atomic.LoadInt32(&resolverCalls)
		require.LessOrEqual(t, actualCalls, int32(10))
		// But at least we should have gotten some caching benefit on subsequent calls
		// Let's do another call to verify caching is working
		result, err := resolver.Resolve(t.Context(), didWeb)
		require.Nil(t, err)
		require.Equal(t, didKey, result)
		// This call should definitely use the cache
		finalCalls := atomic.LoadInt32(&resolverCalls)
		require.Equal(t, actualCalls, finalCalls)
	})

	t.Run("handles different DIDs independently", func(t *testing.T) {
		did1, err := did.Parse("did:web:example1.com")
		require.NoError(t, err)

		did2, err := did.Parse("did:web:example2.com")
		require.NoError(t, err)

		didKey1, err := verifier.Parse("did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK")
		require.NoError(t, err)

		didKey2, err := verifier.Parse("did:key:z6Mkfriq1MqLBoPWecGoDLjguo1sB9brj6wT3qZ5BxkKpuP6")
		require.NoError(t, err)

		mock := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				switch input.String() {
				case did1.String():
					return didKey1, nil
				case did2.String():
					return didKey2, nil
				default:
					return nil, verrs.NewDIDKeyResolutionError(input, fmt.Errorf("unknown DID"))
				}
			},
		}

		resolver, err := didresolver.NewCachedResolver(mock.ResolveDIDKey, 1*time.Second)
		require.NoError(t, err)

		// Resolve first DID
		result1, err1 := resolver.Resolve(t.Context(), did1)
		require.Nil(t, err1)
		require.Equal(t, didKey1, result1)
		require.Equal(t, 1, mock.getCallCount())

		// Resolve second DID
		result2, err2 := resolver.Resolve(t.Context(), did2)
		require.Nil(t, err2)
		require.Equal(t, didKey2, result2)
		require.Equal(t, 2, mock.getCallCount())

		// Resolve first DID again (should be cached)
		result3, err3 := resolver.Resolve(t.Context(), did1)
		require.Nil(t, err3)
		require.Equal(t, didKey1, result3)
		require.Equal(t, 2, mock.getCallCount()) // No additional call

		// Resolve second DID again (should be cached)
		result4, err4 := resolver.Resolve(t.Context(), did2)
		require.Nil(t, err4)
		require.Equal(t, didKey2, result4)
		require.Equal(t, 2, mock.getCallCount()) // No additional call
	})
}

func TestCachedResolver_WithFixedImplementation(t *testing.T) {
	// This test verifies the bug is fixed
	t.Run("wrapped resolver works correctly", func(t *testing.T) {
		didWeb, err := did.Parse("did:web:example.com")
		require.NoError(t, err)

		didKey, err := verifier.Parse("did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK")
		require.NoError(t, err)

		mock := &mockResolver{
			resolveFn: func(ctx context.Context, input did.DID) (ucan.Verifier, error) {
				return didKey, nil
			},
		}

		resolver, err := didresolver.NewCachedResolver(mock.ResolveDIDKey, 1*time.Second)
		require.NoError(t, err)

		// Should not panic and should return the expected result
		result, unresolvedErr := resolver.Resolve(t.Context(), didWeb)
		require.Nil(t, unresolvedErr)
		require.Equal(t, didKey, result)
	})
}

func TestCachedResolver_WithMapResolver(t *testing.T) {
	t.Run("caches MapResolver lookups", func(t *testing.T) {
		// Create a mapping of DIDs
		mapping := map[string]string{
			"did:web:alice.example.com": "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
			"did:web:bob.example.com":   "did:key:z6Mkfriq1MqLBoPWecGoDLjguo1sB9brj6wT3qZ5BxkKpuP6",
			"did:web:carol.example.com": "did:key:z6MkwXG2WjeQnNxSoynSGYU8V9j3QzP3JSqhdmkHc6SaVWoV",
		}

		// Create MapResolver
		mapResolver, err := didresolver.NewMapResolver(mapping)
		require.NoError(t, err)

		// Wrap it with CacheResolver
		cachedResolver, err := didresolver.NewCachedResolver(mapResolver.Resolve, 200*time.Millisecond)
		require.NoError(t, err)

		// Test alice
		aliceDID, err := did.Parse("did:web:alice.example.com")
		require.NoError(t, err)
		aliceKey, err := verifier.Parse("did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK")
		require.NoError(t, err)

		// First call - should hit MapResolver
		result1, err1 := cachedResolver.Resolve(t.Context(), aliceDID)
		require.Nil(t, err1)
		require.Equal(t, aliceKey, result1)

		// Second call - should use cache (we can't directly verify this without instrumentation)
		result2, err2 := cachedResolver.Resolve(t.Context(), aliceDID)
		require.Nil(t, err2)
		require.Equal(t, aliceKey, result2)

		// Test bob while alice is still cached
		bobDID, err := did.Parse("did:web:bob.example.com")
		require.NoError(t, err)
		bobKey, err := verifier.Parse("did:key:z6Mkfriq1MqLBoPWecGoDLjguo1sB9brj6wT3qZ5BxkKpuP6")
		require.NoError(t, err)

		result3, err3 := cachedResolver.Resolve(t.Context(), bobDID)
		require.Nil(t, err3)
		require.Equal(t, bobKey, result3)

		// Wait for cache to expire
		time.Sleep(250 * time.Millisecond)

		// Alice's entry should have expired, this should hit MapResolver again
		result4, err4 := cachedResolver.Resolve(t.Context(), aliceDID)
		require.Nil(t, err4)
		require.Equal(t, aliceKey, result4)

		// Test non-existent DID
		unknownDID, err := did.Parse("did:web:unknown.example.com")
		require.NoError(t, err)

		result5, err5 := cachedResolver.Resolve(t.Context(), unknownDID)
		require.NotNil(t, err5)
		require.Nil(t, result5)
		require.Contains(t, err5.Error(), "unable to resolve")
	})

	t.Run("handles invalid mappings gracefully", func(t *testing.T) {
		// Test with invalid DID in mapping
		invalidMapping := map[string]string{
			"invalid-did": "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
		}

		_, err := didresolver.NewMapResolver(invalidMapping)
		require.Error(t, err)
	})
}
