package didresolver

import (
	"context"
	"fmt"

	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/principal/ed25519/verifier"
	pverifier "github.com/fil-forge/ucantone/principal/verifier"
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
		didKey, err := verifier.Parse(v)
		if err != nil {
			return nil, err
		}
		// token.VerifySignature compares the token's Issuer DID against the
		// verifier's DID. If a did:web (or any non-key DID) maps to a did:key
		// verifier, the equality check fails and signature verification is
		// rejected before the bytes are even examined. Wrap the verifier so
		// it announces the requested DID. did:key inputs are stored unwrapped.
		var dv ucan.Verifier = didKey
		if dk.Method() != "key" {
			wrapped, err := pverifier.Wrap(didKey, dk)
			if err != nil {
				return nil, fmt.Errorf("wrapping verifier as %s: %w", dk, err)
			}
			dv = wrapped
		}
		dmap[dk] = dv
	}
	return &MapResolver{Mapping: dmap}, nil
}
