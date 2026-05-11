package ucanlib

import (
	"context"
	"fmt"
	"iter"

	"github.com/fil-forge/libforge/capabilities/ucan/attest"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/varsig/algorithm/nonstandard"
	"github.com/ipfs/go-cid"
)

// InvocationListerFunc lists invocations that match EXACTLY the given audience,
// command, and subject.
type InvocationListerFunc func(ctx context.Context, aud ucan.Principal, cmd ucan.Command, sub ucan.Subject) iter.Seq2[ucan.Invocation, error]

// ProofAttestations returns a list of attestations for proofs that need them.
// i.e. if a proof is signed with a non-standard signature this function will
// fetch an attestation for it, and fail if it cannot. The authority parameter
// is the DID of the service we trust to be issuing attestations.
func ProofAttestations(ctx context.Context, listInvocations InvocationListerFunc, proofs []ucan.Delegation, authority ucan.Principal) ([]ucan.Invocation, error) {
	var attestations []ucan.Invocation
	for _, proof := range proofs {
		if proof.Signature().Header().SignatureAlgorithm().Code() != nonstandard.Code {
			continue
		}
		var attestation ucan.Invocation
		for inv, err := range listInvocations(ctx, proof.Audience(), attest.ProofCommand, authority) {
			if err != nil {
				return nil, fmt.Errorf("listing invocations for proof signed by %q: %w", proof.Issuer().DID(), err)
			}
			// unlikely since all attestations should be self-signed by the authority
			if inv.Issuer().DID() != authority.DID() {
				continue
			}
			if ucan.IsExpired(inv) {
				continue
			}
			// ensure this attestation corresponds to the proof
			attestedProof, ok := inv.Arguments()["proof"].(cid.Cid)
			if !ok || attestedProof != proof.Link() {
				continue
			}
			attestation = inv
			break
		}
		if attestation == nil {
			return nil, fmt.Errorf("no attestation found for proof signed by %q", proof.Issuer().DID())
		}
		attestations = append(attestations, attestation)
	}
	return attestations, nil
}
