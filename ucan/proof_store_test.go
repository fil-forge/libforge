package ucanlib_test

import (
	"testing"

	"github.com/fil-forge/libforge/capabilities/ucan/attest"
	"github.com/fil-forge/libforge/didmailto"
	"github.com/fil-forge/libforge/testutil"
	ucanlib "github.com/fil-forge/libforge/ucan"
	"github.com/fil-forge/ucantone/principal/absentee"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/ucan/command"
	"github.com/fil-forge/ucantone/ucan/container"
	"github.com/fil-forge/ucantone/ucan/delegation"
	"github.com/fil-forge/ucantone/ucan/invocation"
	"github.com/stretchr/testify/require"
)

func TestContainerProofStore_ProofChain(t *testing.T) {
	t.Run("nil container", func(t *testing.T) {
		ps := ucanlib.NewContainerProofStore(nil)
		space := testutil.RandomSigner(t)
		alice := testutil.Alice
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		proofs, links, err := ps.ProofChain(t.Context(), alice.DID(), cmd, space.DID())
		require.NoError(t, err)
		require.Empty(t, proofs)
		require.Empty(t, links)
	})

	t.Run("empty container", func(t *testing.T) {
		ps := ucanlib.NewContainerProofStore(container.New())
		space := testutil.RandomSigner(t)
		alice := testutil.Alice
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		proofs, links, err := ps.ProofChain(t.Context(), alice.DID(), cmd, space.DID())
		require.NoError(t, err)
		require.Empty(t, proofs)
		require.Empty(t, links)
	})

	t.Run("self-issued root", func(t *testing.T) {
		space := testutil.RandomSigner(t)
		alice := testutil.Alice
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		root := testutil.Must(delegation.Delegate(space, alice.DID(), space.DID(), cmd))(t)

		ct := container.New(container.WithDelegations(root))
		ps := ucanlib.NewContainerProofStore(ct)

		proofs, links, err := ps.ProofChain(t.Context(), alice.DID(), cmd, space.DID())
		require.NoError(t, err)
		assertChain(t, proofs, links, []ucan.Delegation{root})
	})

	t.Run("multi-hop chain", func(t *testing.T) {
		space := testutil.RandomSigner(t)
		alice := testutil.Alice
		bob := testutil.Bob
		carol := testutil.Carol
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		sa := testutil.Must(delegation.Delegate(space, alice.DID(), space.DID(), cmd))(t)
		ab := testutil.Must(delegation.Delegate(alice, bob.DID(), space.DID(), cmd))(t)
		bc := testutil.Must(delegation.Delegate(bob, carol.DID(), space.DID(), cmd))(t)

		ct := container.New(container.WithDelegations(sa, ab, bc))
		ps := ucanlib.NewContainerProofStore(ct)

		proofs, links, err := ps.ProofChain(t.Context(), carol.DID(), cmd, space.DID())
		require.NoError(t, err)
		assertChain(t, proofs, links, []ucan.Delegation{sa, ab, bc})
	})

	t.Run("parent command resolves via matcher", func(t *testing.T) {
		space := testutil.RandomSigner(t)
		alice := testutil.Alice
		parent := testutil.Must(command.Parse("/test"))(t)
		child := testutil.Must(command.Parse("/test/do"))(t)

		root := testutil.Must(delegation.Delegate(space, alice.DID(), space.DID(), parent))(t)

		ct := container.New(container.WithDelegations(root))
		ps := ucanlib.NewContainerProofStore(ct)

		proofs, links, err := ps.ProofChain(t.Context(), alice.DID(), child, space.DID())
		require.NoError(t, err)
		assertChain(t, proofs, links, []ucan.Delegation{root})
	})

	t.Run("broken chain returns empty", func(t *testing.T) {
		space := testutil.RandomSigner(t)
		alice := testutil.Alice
		bob := testutil.Bob
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		// alice → bob exists, but no space → alice root.
		ab := testutil.Must(delegation.Delegate(alice, bob.DID(), space.DID(), cmd))(t)

		ct := container.New(container.WithDelegations(ab))
		ps := ucanlib.NewContainerProofStore(ct)

		proofs, links, err := ps.ProofChain(t.Context(), bob.DID(), cmd, space.DID())
		require.NoError(t, err)
		require.Empty(t, proofs)
		require.Empty(t, links)
	})

	t.Run("filters by audience", func(t *testing.T) {
		space := testutil.RandomSigner(t)
		alice := testutil.Alice
		bob := testutil.Bob
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		// Delegation to alice, but we ask for bob.
		root := testutil.Must(delegation.Delegate(space, alice.DID(), space.DID(), cmd))(t)

		ct := container.New(container.WithDelegations(root))
		ps := ucanlib.NewContainerProofStore(ct)

		proofs, links, err := ps.ProofChain(t.Context(), bob.DID(), cmd, space.DID())
		require.NoError(t, err)
		require.Empty(t, proofs)
		require.Empty(t, links)
	})

	t.Run("filters by subject", func(t *testing.T) {
		spaceA := testutil.RandomSigner(t)
		spaceB := testutil.RandomSigner(t)
		alice := testutil.Alice
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		// Delegation rooted in spaceA, but we ask for spaceB.
		root := testutil.Must(delegation.Delegate(spaceA, alice.DID(), spaceA.DID(), cmd))(t)

		ct := container.New(container.WithDelegations(root))
		ps := ucanlib.NewContainerProofStore(ct)

		proofs, links, err := ps.ProofChain(t.Context(), alice.DID(), cmd, spaceB.DID())
		require.NoError(t, err)
		require.Empty(t, proofs)
		require.Empty(t, links)
	})
}

func TestContainerProofStore_ProofAttestations(t *testing.T) {
	t.Run("nil container with no proofs", func(t *testing.T) {
		ps := ucanlib.NewContainerProofStore(nil)
		service := testutil.WebService

		attestations, err := ps.ProofAttestations(t.Context(), nil, service.DID())
		require.NoError(t, err)
		require.Empty(t, attestations)
	})

	t.Run("standard signatures need no attestations", func(t *testing.T) {
		service := testutil.WebService
		space := testutil.RandomSigner(t)
		alice := testutil.Alice
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		dlg := testutil.Must(delegation.Delegate(space, alice.DID(), space.DID(), cmd))(t)

		ps := ucanlib.NewContainerProofStore(container.New())

		attestations, err := ps.ProofAttestations(t.Context(), []ucan.Delegation{dlg}, service.DID())
		require.NoError(t, err)
		require.Empty(t, attestations)
	})

	t.Run("absentee-signed proof finds attestation in container", func(t *testing.T) {
		service := testutil.WebService
		mailtoDID := testutil.Must(didmailto.New("alice@example.com"))(t)
		account := absentee.From(mailtoDID)
		agent := testutil.Alice
		space := testutil.RandomSigner(t)
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		// account (absentee) → agent — proof needing attestation.
		dlg := testutil.Must(delegation.Delegate(account, agent.DID(), space.DID(), cmd))(t)

		// Authority-issued attestation for the proof.
		att := testutil.Must(attest.Proof.Invoke(
			service,
			service.DID(),
			&attest.ProofArguments{Proof: dlg.Link()},
			invocation.WithAudience(agent.DID()),
		))(t)

		ct := container.New(
			container.WithDelegations(dlg),
			container.WithInvocations(att),
		)
		ps := ucanlib.NewContainerProofStore(ct)

		attestations, err := ps.ProofAttestations(t.Context(), []ucan.Delegation{dlg}, service.DID())
		require.NoError(t, err)
		require.Len(t, attestations, 1)
		require.Equal(t, att.Link(), attestations[0].Link())
	})

	t.Run("absentee-signed proof with missing attestation errors", func(t *testing.T) {
		service := testutil.WebService
		mailtoDID := testutil.Must(didmailto.New("alice@example.com"))(t)
		account := absentee.From(mailtoDID)
		agent := testutil.Alice
		space := testutil.RandomSigner(t)
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		dlg := testutil.Must(delegation.Delegate(account, agent.DID(), space.DID(), cmd))(t)

		ps := ucanlib.NewContainerProofStore(container.New(container.WithDelegations(dlg)))

		attestations, err := ps.ProofAttestations(t.Context(), []ucan.Delegation{dlg}, service.DID())
		require.Error(t, err)
		require.Nil(t, attestations)
	})

	t.Run("attestation lookup filters by audience, command, and subject", func(t *testing.T) {
		service := testutil.WebService
		mailtoDID := testutil.Must(didmailto.New("alice@example.com"))(t)
		account := absentee.From(mailtoDID)
		agent := testutil.Alice
		other := testutil.Bob
		space := testutil.RandomSigner(t)
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		dlg := testutil.Must(delegation.Delegate(account, agent.DID(), space.DID(), cmd))(t)

		// Attestation targeting a different audience — should be ignored.
		wrongAud := testutil.Must(attest.Proof.Invoke(
			service,
			service.DID(),
			&attest.ProofArguments{Proof: dlg.Link()},
			invocation.WithAudience(other.DID()),
		))(t)

		ps := ucanlib.NewContainerProofStore(container.New(
			container.WithDelegations(dlg),
			container.WithInvocations(wrongAud),
		))

		attestations, err := ps.ProofAttestations(t.Context(), []ucan.Delegation{dlg}, service.DID())
		require.Error(t, err)
		require.Nil(t, attestations)
	})
}
