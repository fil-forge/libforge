package didresolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/ucan"
	verrs "github.com/fil-forge/ucantone/validator/errors"
)

// FIXME: remove when https://github.com/fil-forge/ucantone/pull/7 lands
type DIDVerifierResolverFunc func(ctx context.Context, did did.DID) (ucan.Verifier, error)

type TieredResolver struct {
	Tiers []DIDVerifierResolverFunc
}

func (r *TieredResolver) ResolveDIDKey(ctx context.Context, input did.DID) (ucan.Verifier, error) {
	var errs error
	for _, tier := range r.Tiers {
		verifier, err := tier(ctx, input)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		return verifier, nil
	}
	return nil, verrs.NewDIDKeyResolutionError(input, fmt.Errorf("not resolvable by any tier: %w", errs))
}
