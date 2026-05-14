package didresolver

import (
	"context"
	"time"

	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/patrickmn/go-cache"
)

type CachedResolver struct {
	wrapped DIDVerifierResolverFunc
	cache   *cache.Cache
}

func NewCachedResolver(wrapped DIDVerifierResolverFunc, ttl time.Duration) (*CachedResolver, error) {
	// items remain in the cache for `ttl`, expired items are purged every hour.
	return &CachedResolver{wrapped: wrapped, cache: cache.New(ttl, time.Hour)}, nil
}

func (c *CachedResolver) Resolve(ctx context.Context, input did.DID) (ucan.Verifier, error) {
	if out, found := c.cache.Get(input.String()); found {
		return out.(ucan.Verifier), nil
	}
	out, err := c.wrapped(ctx, input)
	if err != nil {
		return nil, err
	}
	c.cache.Set(input.String(), out, cache.DefaultExpiration)

	return out, nil
}
