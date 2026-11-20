package egress_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/fil-forge/automobile"
	"github.com/fil-forge/libforge/commands/content"
	egresscmds "github.com/fil-forge/libforge/commands/egress"
	"github.com/fil-forge/libforge/egress"
	"github.com/fil-forge/libforge/receipt"
	edm "github.com/fil-forge/ucantone/errors/datamodel"
	"github.com/fil-forge/ucantone/execution"
	"github.com/fil-forge/ucantone/testutil"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/ucan/container"
	"github.com/fil-forge/ucantone/ucan/invocation"
	ucanreceipt "github.com/fil-forge/ucantone/ucan/receipt"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
)

// gateway is a principal that serves content from a space it holds a
// two-delegation chain for.
type gateway struct {
	issuer ucan.Issuer
	space  ucan.Issuer
	proofs []ucan.Delegation
}

func newGateway(t *testing.T) *gateway {
	t.Helper()
	space := testutil.RandomIssuer(t)
	key := testutil.RandomIssuer(t)
	issuer := testutil.RandomIssuer(t)
	root := testutil.Must(content.Serve.Delegate(space, key.DID(), space.DID()))(t)
	leaf := testutil.Must(content.Serve.Delegate(key, issuer.DID(), space.DID()))(t)
	return &gateway{issuer: issuer, space: space, proofs: []ucan.Delegation{root, leaf}}
}

// serve returns the record of a `/content/serve` that was asked for size
// stored bytes, read them all and sent sent bytes to the client.
func (g *gateway) serve(t *testing.T, size, sent uint64) egress.Record {
	t.Helper()
	inv := testutil.Must(content.Serve.Invoke(
		g.issuer,
		g.space.DID(),
		&content.ServeArguments{
			Blob:  content.Blob{Digest: testutil.RandomDigest(t)},
			Range: content.Range{Start: 0, End: size - 1},
		},
		invocation.WithAudience(g.issuer.DID()),
		invocation.WithProofs(g.proofs[0].Link(), g.proofs[1].Link()),
	))(t)
	rcpt := testutil.Must(ucanreceipt.IssueOK(g.issuer, inv.Task().Link(), &content.ServeOK{Read: size, Sent: sent}))(t)
	return egress.Record{Receipt: rcpt, Invocation: inv, Proofs: g.proofs}
}

func openJournal(t *testing.T, dir string, options ...egress.JournalOption) *egress.Journal {
	t.Helper()
	j, err := egress.OpenJournal(dir, options...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = j.Close() })
	return j
}

func readBatch(t *testing.T, j *egress.Journal, batch cid.Cid) []egress.Record {
	t.Helper()
	f, err := j.Open(batch)
	require.NoError(t, err)
	defer f.Close()
	var records []egress.Record
	for rec, err := range egress.ReadBatch(f) {
		require.NoError(t, err)
		records = append(records, rec)
	}
	return records
}

func TestJournalSealsARoundTrippableBatch(t *testing.T) {
	g := newGateway(t)
	j := openJournal(t, t.TempDir())
	ctx := context.Background()

	want := []egress.Record{g.serve(t, 100, 100), g.serve(t, 200, 50), g.serve(t, 300, 300)}
	for _, rec := range want {
		require.NoError(t, j.Append(ctx, rec))
	}
	batch, ok, err := j.Seal()
	require.NoError(t, err)
	require.True(t, ok)

	batches, err := j.Batches()
	require.NoError(t, err)
	require.Equal(t, []cid.Cid{batch}, batches)

	got := readBatch(t, j, batch)
	require.Len(t, got, len(want))
	for i, rec := range got {
		require.Equal(t, want[i].Receipt.Link(), rec.Receipt.Link())
		require.Equal(t, want[i].Invocation.Link(), rec.Invocation.Link())
		require.Len(t, rec.Proofs, 2)
		require.Equal(t, g.proofs[0].Link(), rec.Proofs[0].Link())
		require.Equal(t, g.proofs[1].Link(), rec.Proofs[1].Link())
	}

	// The batch is named by the hash of its bytes, and holds each delegation
	// once: two delegations, then an invocation and a receipt per record.
	f, err := j.Open(batch)
	require.NoError(t, err)
	defer f.Close()
	data, err := io.ReadAll(f)
	require.NoError(t, err)
	digest := sha256.Sum256(data)
	mh, err := multihash.Encode(digest[:], multihash.SHA2_256)
	require.NoError(t, err)
	require.Equal(t, cid.NewCidV1(automobile.Code, mh), batch)
	_, blocks, err := automobile.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	require.Len(t, blocks, 2+2*len(want))

	// Nothing has been appended since, so there is nothing to seal.
	_, ok, err = j.Seal()
	require.NoError(t, err)
	require.False(t, ok)
}

func TestJournalSealsAtItsSizeLimit(t *testing.T) {
	g := newGateway(t)
	dir := t.TempDir()
	j := openJournal(t, dir, egress.WithMaxBatchSize(4096))
	ctx := context.Background()

	appended := 0
	for {
		require.NoError(t, j.Append(ctx, g.serve(t, 10, 10)))
		appended++
		batches, err := j.Batches()
		require.NoError(t, err)
		if len(batches) > 0 {
			// The record that would have crossed the limit starts the next
			// batch, so the sealed one is within it.
			require.Len(t, readBatch(t, j, batches[0]), appended-1)
			info, err := os.Stat(filepath.Join(dir, batches[0].String()+".car"))
			require.NoError(t, err)
			require.LessOrEqual(t, info.Size(), int64(4096))
			break
		}
		require.Less(t, appended, 100, "the journal never sealed")
	}
	select {
	case <-j.Sealed():
	default:
		t.Fatal("the journal did not signal the seal")
	}

	// The record held over is in the batch sealed next, with its proofs.
	next, ok, err := j.Seal()
	require.NoError(t, err)
	require.True(t, ok)
	held := readBatch(t, j, next)
	require.Len(t, held, 1)
	require.Len(t, held[0].Proofs, 2)
}

func TestJournalGivesAnOversizedRecordItsOwnBatch(t *testing.T) {
	g := newGateway(t)
	j := openJournal(t, t.TempDir(), egress.WithMaxBatchSize(64))
	ctx := context.Background()

	require.NoError(t, j.Append(ctx, g.serve(t, 10, 10)))
	require.NoError(t, j.Append(ctx, g.serve(t, 10, 10)))
	_, _, err := j.Seal()
	require.NoError(t, err)
	batches, err := j.Batches()
	require.NoError(t, err)
	require.Len(t, batches, 2)
	for _, b := range batches {
		require.Len(t, readBatch(t, j, b), 1)
	}
}

func TestJournalRefusesAnOversizedLimit(t *testing.T) {
	_, err := egress.OpenJournal(t.TempDir(), egress.WithMaxBatchSize(egress.MaxBatchSize+1))
	require.Error(t, err)
}

func TestJournalSealsABatchLeftOpen(t *testing.T) {
	g := newGateway(t)
	dir := t.TempDir()
	ctx := context.Background()

	j, err := egress.OpenJournal(dir)
	require.NoError(t, err)
	want := []egress.Record{g.serve(t, 100, 100), g.serve(t, 200, 200)}
	for _, rec := range want {
		require.NoError(t, j.Append(ctx, rec))
	}
	require.NoError(t, j.Close())

	// A crash part way through a write leaves half a block at the end.
	f, err := os.OpenFile(filepath.Join(dir, "egress.open"), os.O_WRONLY|os.O_APPEND, 0)
	require.NoError(t, err)
	_, err = f.Write([]byte{0xff, 0x01, 0x02, 0x03})
	require.NoError(t, err)
	require.NoError(t, f.Close())

	j = openJournal(t, dir)
	batches, err := j.Batches()
	require.NoError(t, err)
	require.Len(t, batches, 1)
	got := readBatch(t, j, batches[0])
	require.Len(t, got, len(want))
	require.Equal(t, want[1].Receipt.Link(), got[1].Receipt.Link())

	// The trimmed file hashes to the name it was given.
	data, err := os.ReadFile(filepath.Join(dir, batches[0].String()+".car"))
	require.NoError(t, err)
	digest := sha256.Sum256(data)
	mh, err := multihash.Encode(digest[:], multihash.SHA2_256)
	require.NoError(t, err)
	require.Equal(t, cid.NewCidV1(automobile.Code, mh), batches[0])
}

func TestJournalDropsFailedRetrievals(t *testing.T) {
	g := newGateway(t)
	j := openJournal(t, t.TempDir())

	rec := g.serve(t, 100, 100)
	failed, err := ucanreceipt.IssueErr(g.issuer, rec.Invocation.Task().Link(), &edm.ErrorModel{ErrorName: "NotFound", Message: "no such blob"})
	require.NoError(t, err)
	rec.Receipt = failed
	require.NoError(t, j.Append(context.Background(), rec))

	_, ok, err := j.Seal()
	require.NoError(t, err)
	require.False(t, ok)
}

func TestJournalRefusesAnIncompleteRecord(t *testing.T) {
	g := newGateway(t)
	j := openJournal(t, t.TempDir())
	ctx := context.Background()

	missingProof := g.serve(t, 100, 100)
	missingProof.Proofs = missingProof.Proofs[:1]
	require.Error(t, j.Append(ctx, missingProof))

	other := g.serve(t, 100, 100)
	wrongReceipt := g.serve(t, 100, 100)
	wrongReceipt.Receipt = other.Receipt
	require.Error(t, j.Append(ctx, wrongReceipt))
}

func TestHandlerServesSealedBatches(t *testing.T) {
	g := newGateway(t)
	j := openJournal(t, t.TempDir())
	require.NoError(t, j.Append(context.Background(), g.serve(t, 100, 100)))
	batch, _, err := j.Seal()
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.Handle("/.well-known/egress/", egress.NewHandler(j))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/.well-known/egress/" + batch.String())
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, automobile.ContentType, res.Header.Get("Content-Type"))
	n := 0
	for _, err := range egress.ReadBatch(res.Body) {
		require.NoError(t, err)
		n++
	}
	require.Equal(t, 1, n)

	res, err = http.Get(srv.URL + "/.well-known/egress/" + testutil.RandomCID(t).String())
	require.NoError(t, err)
	res.Body.Close()
	require.Equal(t, http.StatusNotFound, res.StatusCode)

	res, err = http.Get(srv.URL + "/.well-known/egress/not-a-cid")
	require.NoError(t, err)
	res.Body.Close()
	require.Equal(t, http.StatusBadRequest, res.StatusCode)
}

// tracker is a fake egress tracking service. It accepts every track, promises
// a consolidation, and issues the consolidation's receipt when told to.
type tracker struct {
	t      *testing.T
	issuer ucan.Issuer

	mu       sync.Mutex
	tracks   []egresscmds.TrackArguments
	promised []ucan.Invocation
	receipts map[cid.Cid]ucan.Receipt
}

func newTracker(t *testing.T) *tracker {
	return &tracker{t: t, issuer: testutil.RandomIssuer(t), receipts: map[cid.Cid]ucan.Receipt{}}
}

func (tr *tracker) Execute(req execution.Request) (execution.Response, error) {
	inv := req.Invocation()
	require.Equal(tr.t, egresscmds.Track.Command.String(), inv.Command().String())
	require.Equal(tr.t, tr.issuer.DID(), inv.Subject())
	var args egresscmds.TrackArguments
	require.NoError(tr.t, args.UnmarshalCBOR(bytes.NewReader(inv.ArgumentsBytes())))

	consolidate, err := egresscmds.Consolidate.Invoke(tr.issuer, tr.issuer.DID(), &egresscmds.ConsolidateArguments{Cause: inv.Task().Link()})
	require.NoError(tr.t, err)

	tr.mu.Lock()
	tr.tracks = append(tr.tracks, args)
	tr.promised = append(tr.promised, consolidate)
	tr.mu.Unlock()

	return execution.NewResponse(
		inv.Task().Link(),
		execution.WithIssuer(tr.issuer),
		execution.WithSuccess(&egresscmds.TrackOK{}),
		execution.WithMetadata(container.New(container.WithInvocations(consolidate))),
	)
}

func (tr *tracker) Fetch(_ context.Context, task cid.Cid) (ucan.Receipt, ucan.Container, error) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	rcpt, ok := tr.receipts[task]
	if !ok {
		return nil, nil, receipt.ErrNotFound
	}
	return rcpt, container.New(container.WithReceipts(rcpt)), nil
}

// consolidate issues the receipt for the latest consolidation promised.
func (tr *tracker) consolidate(ok *egresscmds.ConsolidateOK, failure *edm.ErrorModel) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	task := tr.promised[len(tr.promised)-1].Task().Link()
	var rcpt ucan.Receipt
	var err error
	if failure != nil {
		rcpt, err = ucanreceipt.IssueErr(tr.issuer, task, failure)
	} else {
		rcpt, err = ucanreceipt.IssueOK(tr.issuer, task, ok)
	}
	require.NoError(tr.t, err)
	tr.receipts[task] = rcpt
}

const batchEndpoint = "https://gateway.example.com/.well-known/egress/{cid}"

func newReporter(t *testing.T, j *egress.Journal, g *gateway, tr *tracker, onSettle func(egress.Settlement)) *egress.Reporter {
	t.Helper()
	proof := testutil.Must(egresscmds.Track.Delegate(tr.issuer, g.issuer.DID(), tr.issuer.DID()))(t)
	r, err := egress.NewReporter(egress.ReporterConfig{
		Journal:  j,
		Issuer:   g.issuer,
		Tracker:  tr.issuer.DID(),
		Proofs:   []ucan.Delegation{proof},
		Executor: tr,
		Receipts: tr,
		Endpoint: batchEndpoint,
		OnSettle: onSettle,
	})
	require.NoError(t, err)
	return r
}

func TestReporterTracksThenSettlesABatch(t *testing.T) {
	g := newGateway(t)
	tr := newTracker(t)
	j := openJournal(t, t.TempDir())
	ctx := context.Background()

	require.NoError(t, j.Append(ctx, g.serve(t, 100, 100)))
	require.NoError(t, j.Append(ctx, g.serve(t, 200, 50)))
	batch, _, err := j.Seal()
	require.NoError(t, err)

	var settled []egress.Settlement
	r := newReporter(t, j, g, tr, func(s egress.Settlement) { settled = append(settled, s) })

	// The first pass tracks the batch. The tracker has not consolidated it, so
	// the batch stays.
	require.NoError(t, r.Report(ctx))
	require.Len(t, tr.tracks, 1)
	require.Equal(t, batch, tr.tracks[0].Receipts)
	// The template reaches the tracker as it was configured, placeholder and
	// all, and resolves to the batch's URL.
	require.Equal(t, batchEndpoint, tr.tracks[0].Endpoint)
	batchURL, err := tr.tracks[0].BatchURL()
	require.NoError(t, err)
	require.Equal(t, "https://gateway.example.com/.well-known/egress/"+batch.String(), batchURL.String())
	require.Empty(t, settled)
	batches, err := j.Batches()
	require.NoError(t, err)
	require.Equal(t, []cid.Cid{batch}, batches)

	// A second pass does not track it again.
	require.NoError(t, r.Report(ctx))
	require.Len(t, tr.tracks, 1)

	// Once the consolidation receipt is there, the batch is settled and gone.
	tr.consolidate(&egresscmds.ConsolidateOK{TotalEgress: 150}, nil)
	require.NoError(t, r.Report(ctx))
	require.Equal(t, []egress.Settlement{{Batch: batch, Counted: 150, Consolidated: 150}}, settled)
	batches, err = j.Batches()
	require.NoError(t, err)
	require.Empty(t, batches)
}

func TestReporterRemembersATrackAcrossRestarts(t *testing.T) {
	g := newGateway(t)
	tr := newTracker(t)
	dir := t.TempDir()
	ctx := context.Background()

	j, err := egress.OpenJournal(dir)
	require.NoError(t, err)
	require.NoError(t, j.Append(ctx, g.serve(t, 100, 100)))
	_, _, err = j.Seal()
	require.NoError(t, err)
	require.NoError(t, newReporter(t, j, g, tr, nil).Report(ctx))
	require.NoError(t, j.Close())

	j = openJournal(t, dir)
	var settled []egress.Settlement
	r := newReporter(t, j, g, tr, func(s egress.Settlement) { settled = append(settled, s) })
	tr.consolidate(&egresscmds.ConsolidateOK{TotalEgress: 100}, nil)
	require.NoError(t, r.Report(ctx))
	require.Len(t, tr.tracks, 1)
	require.Len(t, settled, 1)
}

func TestReporterReportsATotalThatDiffers(t *testing.T) {
	g := newGateway(t)
	tr := newTracker(t)
	j := openJournal(t, t.TempDir())
	ctx := context.Background()

	rec := g.serve(t, 100, 100)
	require.NoError(t, j.Append(ctx, rec))
	batch, _, err := j.Seal()
	require.NoError(t, err)

	var settled []egress.Settlement
	r := newReporter(t, j, g, tr, func(s egress.Settlement) { settled = append(settled, s) })
	require.NoError(t, r.Report(ctx))
	rejected := []egresscmds.ReceiptError{{Name: "InvalidProof", Message: "expired", Receipt: rec.Receipt.Link()}}
	tr.consolidate(&egresscmds.ConsolidateOK{TotalEgress: 0, Errors: rejected}, nil)
	require.NoError(t, r.Report(ctx))
	require.Equal(t, []egress.Settlement{{Batch: batch, Counted: 100, Consolidated: 0, Errors: rejected}}, settled)
}

func TestReporterTracksAgainAfterAFailedConsolidation(t *testing.T) {
	g := newGateway(t)
	tr := newTracker(t)
	j := openJournal(t, t.TempDir())
	ctx := context.Background()

	require.NoError(t, j.Append(ctx, g.serve(t, 100, 100)))
	batch, _, err := j.Seal()
	require.NoError(t, err)

	r := newReporter(t, j, g, tr, nil)
	require.NoError(t, r.Report(ctx))
	tr.consolidate(nil, &edm.ErrorModel{ErrorName: "BatchUnavailable", Message: "could not fetch the batch"})
	require.Error(t, r.Report(ctx))

	// The batch is kept and submitted again on the next pass.
	batches, err := j.Batches()
	require.NoError(t, err)
	require.Equal(t, []cid.Cid{batch}, batches)
	require.NoError(t, r.Report(ctx))
	require.Len(t, tr.tracks, 2)
}

func TestMeasureContent(t *testing.T) {
	g := newGateway(t)

	n, err := egress.MeasureContent(g.serve(t, 1000, 250))
	require.NoError(t, err)
	require.Equal(t, uint64(250), n)

	inv, err := content.Retrieve.Invoke(g.issuer, g.space.DID(), &content.RetrieveArguments{
		Blob:  content.Blob{Digest: testutil.RandomDigest(t)},
		Range: content.Range{Start: 10, End: 19},
	})
	require.NoError(t, err)
	rcpt, err := ucanreceipt.IssueOK(g.issuer, inv.Task().Link(), &content.RetrieveOK{})
	require.NoError(t, err)
	n, err = egress.MeasureContent(egress.Record{Receipt: rcpt, Invocation: inv})
	require.NoError(t, err)
	require.Equal(t, uint64(10), n)

	// A record the tracker would refuse is not measured either.
	for _, ok := range []content.ServeOK{
		{Read: 100, Sent: 101},  // more sent than read
		{Read: 1001, Sent: 100}, // more read than the range holds
	} {
		rec := g.serve(t, 1000, 0)
		rec.Receipt, err = ucanreceipt.IssueOK(g.issuer, rec.Invocation.Task().Link(), &ok)
		require.NoError(t, err)
		_, err = egress.MeasureContent(rec)
		require.Error(t, err)
	}

	other, err := egresscmds.Track.Invoke(g.issuer, g.issuer.DID(), &egresscmds.TrackArguments{Receipts: testutil.RandomCID(t)})
	require.NoError(t, err)
	_, err = egress.MeasureContent(egress.Record{Receipt: rcpt, Invocation: other})
	require.Error(t, err)
}

func TestReadBatchReportsAReceiptWithNoInvocation(t *testing.T) {
	g := newGateway(t)
	rec := g.serve(t, 100, 100)
	car := automobile.Encode(nil, []automobile.Block{{Link: rec.Receipt.Link(), Data: rec.Receipt.Bytes()}})
	defer car.Close()

	var errs []error
	for _, err := range egress.ReadBatch(car) {
		errs = append(errs, err)
	}
	require.Len(t, errs, 1)
	require.Error(t, errs[0])
	require.False(t, errors.Is(errs[0], io.EOF))
}

func TestReporterRefusesAnEndpointWithNoPlaceholder(t *testing.T) {
	g := newGateway(t)
	tr := newTracker(t)
	j := openJournal(t, t.TempDir())
	cfg := egress.ReporterConfig{
		Journal:  j,
		Issuer:   g.issuer,
		Tracker:  tr.issuer.DID(),
		Executor: tr,
		Receipts: tr,
	}
	for _, endpoint := range []string{
		"https://gateway.example.com/.well-known/egress/",
		"/.well-known/egress/{cid}",
		"ftp://gateway.example.com/{cid}",
	} {
		cfg.Endpoint = endpoint
		_, err := egress.NewReporter(cfg)
		require.Error(t, err, endpoint)
	}
}

func TestReporterRemovesAConsolidatedBatchItCannotMeasure(t *testing.T) {
	g := newGateway(t)
	tr := newTracker(t)
	j := openJournal(t, t.TempDir())
	ctx := context.Background()

	// The journal takes a record of any command, and the default measure knows
	// only the content ones.
	inv, err := egresscmds.Consolidate.Invoke(g.issuer, g.issuer.DID(), &egresscmds.ConsolidateArguments{Cause: testutil.RandomCID(t)})
	require.NoError(t, err)
	rcpt, err := ucanreceipt.IssueOK(g.issuer, inv.Task().Link(), &egresscmds.ConsolidateOK{})
	require.NoError(t, err)
	require.NoError(t, j.Append(ctx, egress.Record{Receipt: rcpt, Invocation: inv}))
	batch, _, err := j.Seal()
	require.NoError(t, err)

	var settled []egress.Settlement
	r := newReporter(t, j, g, tr, func(s egress.Settlement) { settled = append(settled, s) })
	require.NoError(t, r.Report(ctx))
	tr.consolidate(&egresscmds.ConsolidateOK{TotalEgress: 7}, nil)
	require.NoError(t, r.Report(ctx))

	require.Len(t, settled, 1)
	require.Equal(t, batch, settled[0].Batch)
	require.Error(t, settled[0].CountErr)
	require.Zero(t, settled[0].Counted)
	require.Equal(t, uint64(7), settled[0].Consolidated)
	batches, err := j.Batches()
	require.NoError(t, err)
	require.Empty(t, batches)
}

func TestReporterTracksAgainWhenItsRecordIsUnreadable(t *testing.T) {
	g := newGateway(t)
	tr := newTracker(t)
	dir := t.TempDir()
	j := openJournal(t, dir)
	ctx := context.Background()

	require.NoError(t, j.Append(ctx, g.serve(t, 100, 100)))
	batch, _, err := j.Seal()
	require.NoError(t, err)

	r := newReporter(t, j, g, tr, nil)
	require.NoError(t, r.Report(ctx))
	require.Len(t, tr.tracks, 1)

	// A power cut can leave the record empty.
	record := filepath.Join(dir, batch.String()+".car.track")
	require.NoError(t, os.WriteFile(record, nil, 0o644))
	require.NoError(t, r.Report(ctx))
	require.Len(t, tr.tracks, 2)

	var settled []egress.Settlement
	r = newReporter(t, j, g, tr, func(s egress.Settlement) { settled = append(settled, s) })
	tr.consolidate(&egresscmds.ConsolidateOK{TotalEgress: 100}, nil)
	require.NoError(t, r.Report(ctx))
	require.Len(t, settled, 1)
}

func TestReporterSweepsRecordsWhoseBatchIsGone(t *testing.T) {
	g := newGateway(t)
	tr := newTracker(t)
	dir := t.TempDir()
	j := openJournal(t, dir)
	ctx := context.Background()

	require.NoError(t, j.Append(ctx, g.serve(t, 100, 100)))
	batch, _, err := j.Seal()
	require.NoError(t, err)
	r := newReporter(t, j, g, tr, nil)
	require.NoError(t, r.Report(ctx))

	gone := testutil.RandomCID(t).String()
	orphans := []string{
		filepath.Join(dir, gone+".car.track"),
		filepath.Join(dir, gone+".car.track.tmp"),
		filepath.Join(dir, batch.String()+".car.track.tmp"),
	}
	for _, path := range orphans {
		require.NoError(t, os.WriteFile(path, []byte("left behind"), 0o644))
	}
	require.NoError(t, r.Report(ctx))
	for _, path := range orphans {
		require.NoFileExists(t, path)
	}
	// The record of the batch still held is kept, so it is not tracked again.
	require.FileExists(t, filepath.Join(dir, batch.String()+".car.track"))
	require.Len(t, tr.tracks, 1)
}
