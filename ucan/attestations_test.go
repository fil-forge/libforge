package ucanlib_test

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/fil-forge/libforge/commands/ucan/attest"
	"github.com/fil-forge/libforge/didmailto"
	"github.com/fil-forge/libforge/testutil"
	ucanlib "github.com/fil-forge/libforge/ucan"
	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/principal/absentee"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/ucan/command"
	"github.com/fil-forge/ucantone/ucan/delegation"
	"github.com/fil-forge/ucantone/ucan/invocation"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
)

// recordedCall captures arguments passed to a stub AttestationGetterFunc.
type recordedCall struct {
	aud did.DID
	cmd ucan.Command
	sub did.DID
}

// stubAttestationLister returns an AttestationGetterFunc that produces a fresh
// attestation invocation per call (signed by authority) and records each call.
func stubAttestationLister(authority ucan.Signer, proofs []cid.Cid, calls *[]recordedCall) ucanlib.InvocationListerFunc {
	i := 0
	return func(ctx context.Context, aud did.DID, cmd ucan.Command, sub did.DID) iter.Seq2[ucan.Invocation, error] {
		*calls = append(*calls, recordedCall{aud: aud, cmd: cmd, sub: sub})
		return func(yield func(ucan.Invocation, error) bool) {
			if i >= len(proofs) {
				return
			}
			inv, err := attest.Proof.Invoke(
				authority,
				sub,
				&attest.ProofArguments{Proof: proofs[i]},
				invocation.WithAudience(aud),
			)
			if err != nil {
				yield(nil, err)
				return
			}
			yield(inv, nil)
			i++
		}
	}
}

func TestProofAttestations(t *testing.T) {
	t.Run("no proofs", func(t *testing.T) {
		service := testutil.WebService
		var calls []recordedCall
		lister := stubAttestationLister(service, nil, &calls)

		attestations, err := ucanlib.ProofAttestations(t.Context(), lister, nil, service.DID())
		require.NoError(t, err)
		require.Empty(t, attestations)
		require.Empty(t, calls)
	})

	t.Run("standard signatures only", func(t *testing.T) {
		service := testutil.WebService
		space := testutil.RandomSigner(t)
		alice := testutil.Alice
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		// ed25519-signed proof — should be filtered out (no attestation needed).
		dlg := testutil.Must(delegation.Delegate(space, alice.DID(), space.DID(), cmd))(t)

		var calls []recordedCall
		lister := stubAttestationLister(service, nil, &calls)

		attestations, err := ucanlib.ProofAttestations(t.Context(), lister, []ucan.Delegation{dlg}, service.DID())
		require.NoError(t, err)
		require.Empty(t, attestations)
		require.Empty(t, calls, "lister should not be called for standard signatures")
	})

	t.Run("absentee-signed proof", func(t *testing.T) {
		service := testutil.WebService
		mailtoDID := testutil.Must(didmailto.New("alice@example.com"))(t)
		account := absentee.From(mailtoDID)
		agent := testutil.Alice
		space := testutil.RandomSigner(t)
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		// account (absentee, did:mailto) → agent — this proof needs an attestation.
		dlg := testutil.Must(delegation.Delegate(account, agent.DID(), space.DID(), cmd))(t)

		var calls []recordedCall
		lister := stubAttestationLister(service, []cid.Cid{dlg.Link()}, &calls)

		attestations, err := ucanlib.ProofAttestations(t.Context(), lister, []ucan.Delegation{dlg}, service.DID())
		require.NoError(t, err)
		require.Len(t, attestations, 1)
		require.Len(t, calls, 1)

		// Lister should be called with the proof's audience, the /ucan/attest/proof
		// command, and the authority as subject.
		require.Equal(t, agent.DID(), calls[0].aud)
		require.Equal(t, ucan.Command(attest.Proof), calls[0].cmd)
		require.Equal(t, service.DID(), calls[0].sub)
	})

	t.Run("mixed standard and absentee proofs", func(t *testing.T) {
		service := testutil.WebService
		mailtoDID := testutil.Must(didmailto.New("alice@example.com"))(t)
		account := absentee.From(mailtoDID)
		agent := testutil.Alice
		bob := testutil.Bob
		space := testutil.RandomSigner(t)
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		// standard signature — no attestation needed
		standardDlg := testutil.Must(delegation.Delegate(space, bob.DID(), space.DID(), cmd))(t)
		// absentee signature — needs attestation
		absenteeDlg := testutil.Must(delegation.Delegate(account, agent.DID(), space.DID(), cmd))(t)

		var calls []recordedCall
		lister := stubAttestationLister(service, []cid.Cid{absenteeDlg.Link()}, &calls)

		attestations, err := ucanlib.ProofAttestations(t.Context(), lister, []ucan.Delegation{standardDlg, absenteeDlg}, service.DID())
		require.NoError(t, err)
		require.Len(t, attestations, 1, "only the absentee-signed proof needs an attestation")
		require.Len(t, calls, 1)
		require.Equal(t, agent.DID(), calls[0].aud)
	})

	t.Run("multiple absentee-signed proofs", func(t *testing.T) {
		service := testutil.WebService
		aliceMailto := testutil.Must(didmailto.New("alice@example.com"))(t)
		bobMailto := testutil.Must(didmailto.New("bob@example.com"))(t)
		aliceAccount := absentee.From(aliceMailto)
		bobAccount := absentee.From(bobMailto)

		agentA := testutil.Alice
		agentB := testutil.Bob
		space := testutil.RandomSigner(t)
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		dlgA := testutil.Must(delegation.Delegate(aliceAccount, agentA.DID(), space.DID(), cmd))(t)
		dlgB := testutil.Must(delegation.Delegate(bobAccount, agentB.DID(), space.DID(), cmd))(t)

		var calls []recordedCall
		lister := stubAttestationLister(service, []cid.Cid{dlgA.Link(), dlgB.Link()}, &calls)

		attestations, err := ucanlib.ProofAttestations(t.Context(), lister, []ucan.Delegation{dlgA, dlgB}, service.DID())
		require.NoError(t, err)
		require.Len(t, attestations, 2)
		require.Len(t, calls, 2)
		require.Equal(t, agentA.DID(), calls[0].aud)
		require.Equal(t, agentB.DID(), calls[1].aud)
	})

	t.Run("lister error is propagated", func(t *testing.T) {
		service := testutil.WebService
		mailtoDID := testutil.Must(didmailto.New("alice@example.com"))(t)
		account := absentee.From(mailtoDID)
		agent := testutil.Alice
		space := testutil.RandomSigner(t)
		cmd := testutil.Must(command.Parse("/test/do"))(t)

		dlg := testutil.Must(delegation.Delegate(account, agent.DID(), space.DID(), cmd))(t)

		wantErr := errors.New("boom")
		lister := func(ctx context.Context, aud did.DID, cmd ucan.Command, sub did.DID) iter.Seq2[ucan.Invocation, error] {
			return func(yield func(ucan.Invocation, error) bool) {
				yield(nil, wantErr)
			}
		}

		attestations, err := ucanlib.ProofAttestations(t.Context(), lister, []ucan.Delegation{dlg}, service.DID())
		require.ErrorIs(t, err, wantErr)
		require.Nil(t, attestations)
	})
}
