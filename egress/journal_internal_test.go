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
	rcpt, err := receipt.IssueOK(issuer, inv.Task().Link(), &content.ServeOK{Bytes: 10})
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
