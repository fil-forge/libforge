package ucanlib_test

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/fil-forge/libforge/testutil"
	ucanlib "github.com/fil-forge/libforge/ucan"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/ucan/command"
	"github.com/fil-forge/ucantone/ucan/delegation"
	"github.com/stretchr/testify/require"
)

// memLister is an in-memory [ucanlib.DelegationLister] for tests.
type memLister struct {
	delegations []ucan.Delegation
}

func (f *memLister) List(ctx context.Context, aud ucan.Principal, cmd ucan.Command, sub ucan.Subject) iter.Seq2[ucan.Delegation, error] {
	return func(yield func(ucan.Delegation, error) bool) {
		for _, d := range f.delegations {
			if d.Audience().DID() != aud.DID() {
				continue
			}
			if d.Command() != cmd {
				continue
			}
			if sub == nil {
				if d.Subject() != nil {
					continue
				}
			} else {
				if d.Subject() == nil || d.Subject().DID() != sub.DID() {
					continue
				}
			}
			if !yield(d, nil) {
				return
			}
		}
	}
}

// erroringLister yields a single error from the iterator.
type erroringLister struct{ err error }

func (f *erroringLister) List(ctx context.Context, aud ucan.Principal, cmd ucan.Command, sub ucan.Subject) iter.Seq2[ucan.Delegation, error] {
	return func(yield func(ucan.Delegation, error) bool) {
		yield(nil, f.err)
	}
}

func assertChain(t *testing.T, proofs []ucan.Delegation, links []ucan.Link, want []ucan.Delegation) {
	t.Helper()
	require.Len(t, proofs, len(want), "proof chain length")
	require.Len(t, links, len(want), "link chain length")
	for i, w := range want {
		require.Equal(t, w.Link(), proofs[i].Link(), "proofs[%d] link", i)
		require.Equal(t, w.Link(), links[i], "links[%d]", i)
	}
}

func TestProofChain_SelfIssued(t *testing.T) {
	space := testutil.RandomSigner(t)
	alice := testutil.Alice
	cmd := testutil.Must(command.Parse("/test/do"))(t)

	// space delegates to alice (root of chain, subject is the space).
	root := testutil.Must(delegation.Delegate(space, alice, space, cmd))(t)

	finder := &memLister{delegations: []ucan.Delegation{root}}
	matcher := ucanlib.NewDelegationMatcher(finder.List)

	proofs, links, err := ucanlib.ProofChain(t.Context(), matcher, alice, cmd, space)
	require.NoError(t, err)
	assertChain(t, proofs, links, []ucan.Delegation{root})
}

func TestProofChain_MultiHop(t *testing.T) {
	space := testutil.RandomSigner(t)
	alice := testutil.Alice
	bob := testutil.Bob
	carol := testutil.Carol
	cmd := testutil.Must(command.Parse("/test/do"))(t)

	// space → alice (root, subject is the space)
	sa := testutil.Must(delegation.Delegate(space, alice, space, cmd))(t)
	// alice → bob (re-delegates the space's authority)
	ab := testutil.Must(delegation.Delegate(alice, bob, space, cmd))(t)
	// bob → carol (re-delegates the space's authority)
	bc := testutil.Must(delegation.Delegate(bob, carol, space, cmd))(t)

	finder := &memLister{delegations: []ucan.Delegation{sa, ab, bc}}
	matcher := ucanlib.NewDelegationMatcher(finder.List)

	proofs, links, err := ucanlib.ProofChain(t.Context(), matcher, carol, cmd, space)
	require.NoError(t, err)
	// Expected order: root first, then in sequence so aud of prev = iss of next.
	assertChain(t, proofs, links, []ucan.Delegation{sa, ab, bc})
}

func TestProofChain_NoDelegations(t *testing.T) {
	space := testutil.RandomSigner(t)
	alice := testutil.Alice
	cmd := testutil.Must(command.Parse("/test/do"))(t)

	matcher := ucanlib.NewDelegationMatcher((&memLister{}).List)
	proofs, links, err := ucanlib.ProofChain(t.Context(), matcher, alice, cmd, space)
	require.NoError(t, err)
	require.Empty(t, proofs)
	require.Empty(t, links)
}

func TestProofChain_BrokenChain(t *testing.T) {
	space := testutil.RandomSigner(t)
	alice := testutil.Alice
	bob := testutil.Bob
	cmd := testutil.Must(command.Parse("/test/do"))(t)

	// alice → bob exists, but no space → alice root.
	ab := testutil.Must(delegation.Delegate(alice, bob, space, cmd))(t)

	finder := &memLister{delegations: []ucan.Delegation{ab}}
	matcher := ucanlib.NewDelegationMatcher(finder.List)

	proofs, links, err := ucanlib.ProofChain(t.Context(), matcher, bob, cmd, space)
	require.NoError(t, err)
	require.Empty(t, proofs)
	require.Empty(t, links)
}

func TestProofChain_ParentCommand(t *testing.T) {
	space := testutil.RandomSigner(t)
	alice := testutil.Alice
	parent := testutil.Must(command.Parse("/test"))(t)
	child := testutil.Must(command.Parse("/test/do"))(t)

	// space delegates to alice for the parent command.
	root := testutil.Must(delegation.Delegate(space, alice, space, parent))(t)

	finder := &memLister{delegations: []ucan.Delegation{root}}
	matcher := ucanlib.NewDelegationMatcher(finder.List)

	// Invocation for the child command should still resolve via the parent.
	proofs, links, err := ucanlib.ProofChain(t.Context(), matcher, alice, child, space)
	require.NoError(t, err)
	assertChain(t, proofs, links, []ucan.Delegation{root})
}

func TestProofChain_Powerline(t *testing.T) {
	space := testutil.RandomSigner(t)
	alice := testutil.Alice
	bob := testutil.Bob
	cmd := testutil.Must(command.Parse("/test/do"))(t)

	// space delegates to alice (root).
	root := testutil.Must(delegation.Delegate(space, alice, space, cmd))(t)
	// powerline: alice → bob with nil subject.
	powerline := testutil.Must(delegation.Delegate(alice, bob, nil, cmd))(t)

	finder := &memLister{delegations: []ucan.Delegation{root, powerline}}
	matcher := ucanlib.NewDelegationMatcher(finder.List)

	proofs, links, err := ucanlib.ProofChain(t.Context(), matcher, bob, cmd, space)
	require.NoError(t, err)
	assertChain(t, proofs, links, []ucan.Delegation{root, powerline})
}

func TestProofChain_UnrelatedCommandIgnored(t *testing.T) {
	space := testutil.RandomSigner(t)
	alice := testutil.Alice
	cmd := testutil.Must(command.Parse("/test/do"))(t)
	other := testutil.Must(command.Parse("/other/op"))(t)

	// delegation exists but for an unrelated command path.
	dlg := testutil.Must(delegation.Delegate(space, alice, space, other))(t)

	finder := &memLister{delegations: []ucan.Delegation{dlg}}
	matcher := ucanlib.NewDelegationMatcher(finder.List)

	proofs, links, err := ucanlib.ProofChain(t.Context(), matcher, alice, cmd, space)
	require.NoError(t, err)
	require.Empty(t, proofs)
	require.Empty(t, links)
}

func TestProofChain_FinderError(t *testing.T) {
	space := testutil.RandomSigner(t)
	alice := testutil.Alice
	cmd := testutil.Must(command.Parse("/test/do"))(t)

	wantErr := errors.New("boom")
	matcher := ucanlib.NewDelegationMatcher(
		func(ctx context.Context, aud ucan.Principal, cmd ucan.Command, sub ucan.Subject) iter.Seq2[ucan.Delegation, error] {
			return func(yield func(ucan.Delegation, error) bool) {
				yield(nil, wantErr)
			}
		},
	)

	_, _, err := ucanlib.ProofChain(t.Context(), matcher, alice, cmd, space)
	require.ErrorIs(t, err, wantErr)
}
