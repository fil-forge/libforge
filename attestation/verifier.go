package attestation

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"

	"github.com/fil-forge/libforge/commands/ucan/attest"
	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/ucan/invocation"
	"github.com/fil-forge/ucantone/validator"
)

// Verifier is a ucan.Verifier for a DID whose signing is attested by an
// authority verifier.
type Verifier struct {
	ctx               context.Context
	authorityID       did.DID
	authorityVerifier ucan.Verifier
}

var _ ucan.Verifier = Verifier{}

func (v Verifier) String() string {
	return fmt.Sprintf("Attested Verifier{authority=%s}", v.authorityID)
}

func (v Verifier) Verify(msg []byte, sig []byte) bool {
	inv, err := invocation.Decode(sig)
	if err != nil {
		return false
	}

	var args attest.ProofArguments
	err = args.UnmarshalCBOR(bytes.NewReader(inv.ArgumentsBytes()))
	if err != nil {
		return false
	}

	msgDigest, err := mh.Sum(msg, mh.SHA2_256, -1)
	if err != nil {
		return false
	}

	if args.Proof != cid.NewCidV1(cid.Raw, msgDigest) {
		return false
	}

	if inv.Subject() != v.authorityID {
		return false
	}

	if validator.ValidateInvocation(v.ctx, inv) != nil {
		return false
	}

	return true
}

func AttestedVerifier(ctx context.Context, authorityID did.DID, authority ucan.Verifier) ucan.Verifier {
	return Verifier{
		ctx:               ctx,
		authorityID:       authorityID,
		authorityVerifier: authority,
	}
}
