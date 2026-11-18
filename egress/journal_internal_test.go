package egress

import (
	"context"
	"testing"
	"time"

	"github.com/fil-forge/libforge/commands/content"
	"github.com/fil-forge/ucantone/testutil"
	"github.com/fil-forge/ucantone/ucan/receipt"
	"github.com/stretchr/testify/require"
)

func TestJournalSealsAtItsAgeLimit(t *testing.T) {
	now := time.Now()
	// A flush interval this long keeps the journal's own ticker out of the
	// test, which drives tick by hand.
	j, err := OpenJournal(t.TempDir(), WithMaxBatchAge(time.Hour), WithFlushInterval(24*time.Hour))
	require.NoError(t, err)
	t.Cleanup(func() { _ = j.Close() })
	j.cfg.now = func() time.Time { return now }

	issuer := testutil.RandomIssuer(t)
	inv, err := content.Serve.Invoke(issuer, issuer.DID(), &content.ServeArguments{
		Blob:  content.Blob{Digest: testutil.RandomDigest(t)},
		Range: content.Range{Start: 0, End: 9},
	})
	require.NoError(t, err)
	rcpt, err := receipt.IssueOK(issuer, inv.Task().Link(), &content.ServeOK{Read: 10, Sent: 10})
	require.NoError(t, err)
	require.NoError(t, j.Append(context.Background(), Record{Receipt: rcpt, Invocation: inv}))

	// Short of the limit, a tick only flushes.
	now = now.Add(time.Hour - time.Second)
	require.NoError(t, j.tick())
	batches, err := j.Batches()
	require.NoError(t, err)
	require.Empty(t, batches)

	now = now.Add(time.Second)
	require.NoError(t, j.tick())
	batches, err = j.Batches()
	require.NoError(t, err)
	require.Len(t, batches, 1)

	// With no open batch there is nothing to age.
	now = now.Add(2 * time.Hour)
	require.NoError(t, j.tick())
	batches, err = j.Batches()
	require.NoError(t, err)
	require.Len(t, batches, 1)
}

func TestJournalCarriesOnAfterAFailedWrite(t *testing.T) {
	j, err := OpenJournal(t.TempDir(), WithFlushInterval(24*time.Hour))
	require.NoError(t, err)
	t.Cleanup(func() { _ = j.Close() })
	ctx := context.Background()
	issuer := testutil.RandomIssuer(t)
	record := func() Record {
		inv, err := content.Serve.Invoke(issuer, issuer.DID(), &content.ServeArguments{
			Blob:  content.Blob{Digest: testutil.RandomDigest(t)},
			Range: content.Range{Start: 0, End: 9},
		})
		require.NoError(t, err)
		rcpt, err := receipt.IssueOK(issuer, inv.Task().Link(), &content.ServeOK{Read: 10, Sent: 10})
		require.NoError(t, err)
		return Record{Receipt: rcpt, Invocation: inv}
	}

	// One record reaches the disk, then the file fails under the journal, as
	// it would on a full disk, with a second record still buffered.
	first := record()
	require.NoError(t, j.Append(ctx, first))
	require.NoError(t, j.tick())
	require.NoError(t, j.file.Close())
	require.NoError(t, j.Append(ctx, record()))
	require.Error(t, j.tick())

	// The next append works, and what reached the disk is sealed, not lost.
	third := record()
	require.NoError(t, j.Append(ctx, third))
	batches, err := j.Batches()
	require.NoError(t, err)
	require.Len(t, batches, 1)
	f, err := j.Open(batches[0])
	require.NoError(t, err)
	var got []Record
	for rec, err := range ReadBatch(f) {
		require.NoError(t, err)
		got = append(got, rec)
	}
	require.NoError(t, f.Close())
	require.Len(t, got, 1)
	require.Equal(t, first.Receipt.Link(), got[0].Receipt.Link())

	next, ok, err := j.Seal()
	require.NoError(t, err)
	require.True(t, ok)
	f, err = j.Open(next)
	require.NoError(t, err)
	got = got[:0]
	for rec, err := range ReadBatch(f) {
		require.NoError(t, err)
		got = append(got, rec)
	}
	require.NoError(t, f.Close())
	require.Len(t, got, 1)
	require.Equal(t, third.Receipt.Link(), got[0].Receipt.Link())
}

func TestJournalKeepsABatchItCouldNotSeal(t *testing.T) {
	j, err := OpenJournal(t.TempDir(), WithFlushInterval(24*time.Hour))
	require.NoError(t, err)
	t.Cleanup(func() { _ = j.Close() })
	ctx := context.Background()
	issuer := testutil.RandomIssuer(t)
	inv, err := content.Serve.Invoke(issuer, issuer.DID(), &content.ServeArguments{
		Blob:  content.Blob{Digest: testutil.RandomDigest(t)},
		Range: content.Range{Start: 0, End: 9},
	})
	require.NoError(t, err)
	rcpt, err := receipt.IssueOK(issuer, inv.Task().Link(), &content.ServeOK{Read: 10, Sent: 10})
	require.NoError(t, err)
	require.NoError(t, j.Append(ctx, Record{Receipt: rcpt, Invocation: inv}))
	require.NoError(t, j.tick())

	// The seal's close fails after its records are on disk.
	require.NoError(t, j.file.Close())
	_, _, err = j.Seal()
	require.Error(t, err)

	// The next append seals the batch left behind before starting its own.
	inv2, err := content.Serve.Invoke(issuer, issuer.DID(), &content.ServeArguments{
		Blob:  content.Blob{Digest: testutil.RandomDigest(t)},
		Range: content.Range{Start: 0, End: 9},
	})
	require.NoError(t, err)
	rcpt2, err := receipt.IssueOK(issuer, inv2.Task().Link(), &content.ServeOK{Read: 10, Sent: 10})
	require.NoError(t, err)
	require.NoError(t, j.Append(ctx, Record{Receipt: rcpt2, Invocation: inv2}))
	batches, err := j.Batches()
	require.NoError(t, err)
	require.Len(t, batches, 1)
	f, err := j.Open(batches[0])
	require.NoError(t, err)
	defer f.Close()
	n := 0
	for rec, err := range ReadBatch(f) {
		require.NoError(t, err)
		require.Equal(t, rcpt.Link(), rec.Receipt.Link())
		n++
	}
	require.Equal(t, 1, n)
}
