package didresolver

import (
	"context"
	"fmt"

	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/principal/ed25519/verifier"
	"github.com/fil-forge/ucantone/ucan"
	verrs "github.com/fil-forge/ucantone/validator/errors"
)

type MapResolver struct {
	Mapping map[did.DID]ucan.Verifier
}

func (r *MapResolver) Resolve(_ context.Context, input did.DID) (ucan.Verifier, error) {
	// ctx is unused; this implementation only looks in a local mapping.
	dk, ok := r.Mapping[input]
	if !ok {
		return nil, verrs.NewDIDKeyResolutionError(input, fmt.Errorf("not found in mapping: %s", input))
	}
	return dk, nil
}

// NewMapResolver creates a new MapResolver from a mapping of DID string to
// verifier string.
func NewMapResolver(smap map[string]string) (*MapResolver, error) {
	dmap := map[did.DID]ucan.Verifier{}
	for k, v := range smap {
		dk, err := did.Parse(k)
		if err != nil {
			return nil, err
		}
		// TODO: multiple verification methods when https://github.com/fil-forge/ucantone/pull/7 lands
		dv, err := verifier.Parse(v)
		if err != nil {
			return nil, err
		}
		dmap[dk] = dv
	}
	return &MapResolver{Mapping: dmap}, nil
}
