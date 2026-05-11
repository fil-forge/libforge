package ucanlib

import (
	"context"
	"iter"
	"slices"

	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/ucan/command"
)

// DelegationMatcherFunc finds all delegations matching the given audience,
// command, and subject.
//
// The subject parameter MUST not be nil, but matching delegations MAY include
// powerline delegations (with nil subject) and delegations where command is a
// matching parent of the passed command e.g. if passed command is "/read/file",
// delegations with command "/read", and "/" may be returned.
type DelegationMatcherFunc func(ctx context.Context, aud ucan.Principal, cmd ucan.Command, sub ucan.Subject) iter.Seq2[ucan.Delegation, error]

// DelegationListerFunc lists delegations for the given audience, command, and
// subject. It differs from [DelegationMatcherFunc] in that it only retrieves
// delegations for the EXACT audience, command and subject.
//
// Note: the subject parameter MAY be nil to indicate powerline.
type DelegationListerFunc func(ctx context.Context, aud ucan.Principal, cmd ucan.Command, sub ucan.Subject) iter.Seq2[ucan.Delegation, error]

// NewDelegationMatcher creates a simple delegation matcher that queries the
// passed finder to retrieve delegations matching the given audience, command,
// and subject.
func NewDelegationMatcher(listDelegations DelegationListerFunc) DelegationMatcherFunc {
	return func(ctx context.Context, aud ucan.Principal, cmd ucan.Command, sub ucan.Principal) iter.Seq2[ucan.Delegation, error] {
		return func(yield func(ucan.Delegation, error) bool) {
			cmdVariations := []ucan.Command{}
			segs := cmd.Segments()
			for i := len(segs) - 1; i >= 0; i-- {
				cmd := command.Top().Join(segs[0 : i+1]...)
				cmdVariations = append(cmdVariations, cmd)
			}
			cmdVariations = append(cmdVariations, command.Top())

			for _, cmd := range cmdVariations {
				for dlg, err := range listDelegations(ctx, aud, cmd, sub) {
					if err != nil {
						yield(nil, err)
						return
					}
					if !yield(dlg, nil) {
						return
					}
				}
				// try powerline
				// TODO: stop early if we already found delegations?
				for dlg, err := range listDelegations(ctx, aud, cmd, nil) {
					if err != nil {
						yield(nil, err)
						return
					}
					if !yield(dlg, nil) {
						return
					}
				}
			}
		}
	}
}

// ProofChain recursively builds a proof chain of delegations from the given
// audience to the given subject for the specified command. It returns the list
// of delegations and their corresponding links in the order required for
// invocation. i.e. starting from the root Delegation (issued by the Subject),
// in strict sequence where the aud of the previous Delegation matches the iss
// of the next Delegation.
func ProofChain(ctx context.Context, matchDelegations DelegationMatcherFunc, aud ucan.Principal, cmd ucan.Command, sub ucan.Principal) ([]ucan.Delegation, []ucan.Link, error) {
	proofs, links, err := proofChain(ctx, matchDelegations, aud, cmd, sub)
	if err != nil {
		return nil, nil, err
	}
	slices.Reverse(proofs)
	slices.Reverse(links)
	return proofs, links, nil
}

// proofChain returns the delegations and links from the audience toward the
// subject, i.e. in reverse of the invocation order. [ProofChain] reverses the
// result before returning it to the caller.
func proofChain(ctx context.Context, matchDelegations DelegationMatcherFunc, aud ucan.Principal, cmd ucan.Command, sub ucan.Principal) ([]ucan.Delegation, []ucan.Link, error) {
	var proofs []ucan.Delegation
	var links []ucan.Link

	for d, err := range matchDelegations(ctx, aud, cmd, sub) {
		if err != nil {
			return nil, nil, err
		}
		if d.Subject() != nil && d.Subject().DID() == d.Issuer().DID() {
			proofs = append(proofs, d)
			links = append(links, d.Link())
			break
		}
		// if subject is nil, or subject != issuer, we need more proof
		ps, ls, err := proofChain(ctx, matchDelegations, d.Issuer(), d.Command(), sub)
		if err != nil {
			return nil, nil, err
		}
		if len(ps) == 0 {
			continue // try a different path
		}
		proofs = append(proofs, d)
		proofs = append(proofs, ps...)
		links = append(links, d.Link())
		links = append(links, ls...)
		break
	}

	return proofs, links, nil
}
