package egress

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"time"

	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/libforge/commands/content"
	egresscmds "github.com/fil-forge/libforge/commands/egress"
	"github.com/fil-forge/libforge/receipt"
	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/execution"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/ucan/container"
	"github.com/fil-forge/ucantone/ucan/invocation"
	"github.com/ipfs/go-cid"
)

// DefaultReportInterval is how often a [Reporter] looks for batches to track
// and consolidation receipts to collect, when no batch has just been sealed.
const DefaultReportInterval = time.Minute

const trackSuffix = ".track"

// ReceiptFetcher fetches the receipt for a task from the egress tracking
// service. It returns [receipt.ErrNotFound] while the task has not run.
// [receipt.Client] implements it.
type ReceiptFetcher interface {
	Fetch(ctx context.Context, task cid.Cid) (ucan.Receipt, ucan.Container, error)
}

// Settlement is the outcome of a batch the tracker has consolidated.
type Settlement struct {
	// Batch is the CID of the batch.
	Batch cid.Cid
	// Counted is the egress of the batch by the reporter's own measure.
	Counted uint64
	// Consolidated is the egress the tracker's consolidation receipt reports.
	Consolidated uint64
	// Errors are the receipts the tracker could not process.
	Errors []egresscmds.ReceiptError
}

// ReporterConfig configures a [Reporter].
type ReporterConfig struct {
	// Journal holds the batches to report.
	Journal *Journal
	// Issuer signs the `/egress/track` invocations. It must be the issuer of
	// every receipt in the journal.
	Issuer ucan.Issuer
	// Tracker is the egress tracking service.
	Tracker did.DID
	// Proofs is the delegation chain that lets Issuer invoke `/egress/track`
	// on Tracker.
	Proofs []ucan.Delegation
	// Executor sends invocations to the tracker.
	Executor execution.Executor
	// Receipts fetches consolidation receipts from the tracker.
	Receipts ReceiptFetcher
	// Endpoint is the URL the tracker fetches batches from, with a `{cid}`
	// placeholder for the batch CID. It must reach the handler [NewHandler]
	// returns for Journal.
	Endpoint *url.URL
	// Interval is how often to look for work when no batch has just been
	// sealed. The default is [DefaultReportInterval].
	Interval time.Duration
	// Measure returns the egress a record accounts for, which is summed over a
	// batch and compared with the tracker's total. The default is
	// [MeasureContent].
	Measure func(Record) (uint64, error)
	// OnSettle, when set, is called with the outcome of each batch the tracker
	// has consolidated, before the batch is removed.
	OnSettle func(Settlement)
}

// Reporter submits sealed batches to the egress tracking service and removes
// each one once the tracker has consolidated it.
//
// What the reporter knows about a batch is kept in the journal's directory
// beside it, so a restart picks up where the last process stopped.
type Reporter struct {
	cfg ReporterConfig
}

// NewReporter creates a reporter. Call [Reporter.Run] to start it.
func NewReporter(cfg ReporterConfig) (*Reporter, error) {
	switch {
	case cfg.Journal == nil:
		return nil, errors.New("reporter needs a journal")
	case cfg.Issuer == nil:
		return nil, errors.New("reporter needs an issuer")
	case cfg.Tracker == did.Undef:
		return nil, errors.New("reporter needs the tracker's DID")
	case cfg.Executor == nil:
		return nil, errors.New("reporter needs an executor")
	case cfg.Receipts == nil:
		return nil, errors.New("reporter needs a receipt fetcher")
	case cfg.Endpoint == nil:
		return nil, errors.New("reporter needs the batch endpoint")
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultReportInterval
	}
	if cfg.Measure == nil {
		cfg.Measure = MeasureContent
	}
	return &Reporter{cfg: cfg}, nil
}

// Run reports batches until ctx is done. It works through the journal's
// batches whenever one is sealed and at every interval.
func (r *Reporter) Run(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()
	for {
		if err := r.Report(ctx); err != nil && ctx.Err() == nil {
			log.Errorw("reporting egress batches", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-r.cfg.Journal.Sealed():
		}
	}
}

// Report makes one pass over the journal's sealed batches: it tracks each
// batch the tracker has not accepted yet, and settles each batch whose
// consolidation receipt has arrived. It returns the errors of the batches it
// could not move forward, having tried them all.
func (r *Reporter) Report(ctx context.Context) error {
	batches, err := r.cfg.Journal.Batches()
	if err != nil {
		return err
	}
	var errs error
	for _, batch := range batches {
		if ctx.Err() != nil {
			return errors.Join(errs, ctx.Err())
		}
		tracked, err := r.readTracked(batch)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("batch %s: %w", batch, err))
			continue
		}
		if tracked == nil {
			if tracked, err = r.track(ctx, batch); err != nil {
				errs = errors.Join(errs, fmt.Errorf("tracking batch %s: %w", batch, err))
				continue
			}
		}
		if err := r.settle(ctx, batch, tracked); err != nil {
			errs = errors.Join(errs, fmt.Errorf("settling batch %s: %w", batch, err))
		}
	}
	return errs
}

// tracked is what the tracker's answer to `/egress/track` leaves the reporter
// to remember about a batch.
type tracked struct {
	track       ucan.Invocation
	consolidate ucan.Invocation
}

// track invokes `/egress/track` for the batch and records the consolidation
// the tracker promises.
func (r *Reporter) track(ctx context.Context, batch cid.Cid) (*tracked, error) {
	links := make([]cid.Cid, 0, len(r.cfg.Proofs))
	for _, p := range r.cfg.Proofs {
		links = append(links, p.Link())
	}
	inv, err := egresscmds.Track.Invoke(
		r.cfg.Issuer,
		r.cfg.Tracker,
		&egresscmds.TrackArguments{
			Receipts: batch,
			Endpoint: commands.CborURL(*r.cfg.Endpoint),
		},
		invocation.WithProofs(links...),
	)
	if err != nil {
		return nil, fmt.Errorf("issuing invocation: %w", err)
	}
	res, err := r.cfg.Executor.Execute(execution.NewRequest(ctx, inv, execution.WithDelegations(r.cfg.Proofs...)))
	if err != nil {
		return nil, fmt.Errorf("executing invocation: %w", err)
	}
	meta := res.Metadata()
	if closer, ok := meta.(io.Closer); ok {
		defer closer.Close()
	}
	if _, err := egresscmds.Track.Unpack(res.Receipt()); err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, errors.New("the response carries no consolidation task")
	}
	consolidate, err := findConsolidate(meta.Invocations(), inv.Task().Link())
	if err != nil {
		return nil, err
	}
	t := &tracked{track: inv, consolidate: consolidate}
	if err := r.writeTracked(batch, t); err != nil {
		return nil, err
	}
	log.Infow("tracked egress batch", "batch", batch, "consolidate", consolidate.Task().Link())
	return t, nil
}

// settle collects the batch's consolidation receipt, if the tracker has
// issued it, checks the total and removes the batch.
func (r *Reporter) settle(ctx context.Context, batch cid.Cid, t *tracked) error {
	rcpt, ct, err := r.cfg.Receipts.Fetch(ctx, t.consolidate.Task().Link())
	if closer, ok := ct.(io.Closer); ok {
		defer closer.Close()
	}
	if errors.Is(err, receipt.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("fetching consolidation receipt: %w", err)
	}
	if rcpt.Issuer() != t.consolidate.Subject() {
		return fmt.Errorf("consolidation receipt is issued by %s, not %s", rcpt.Issuer(), t.consolidate.Subject())
	}
	ok, err := egresscmds.Consolidate.Unpack(rcpt)
	if err != nil {
		// The tracker could not process the batch at all. Forget the track so
		// the next pass submits the batch again.
		if rmErr := os.Remove(r.trackedPath(batch)); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
			err = errors.Join(err, rmErr)
		}
		return err
	}
	counted, err := r.count(batch)
	if err != nil {
		return fmt.Errorf("measuring batch: %w", err)
	}
	if counted != ok.TotalEgress || len(ok.Errors) > 0 {
		log.Warnw("consolidated egress differs from the batch",
			"batch", batch, "counted", counted, "consolidated", ok.TotalEgress, "errors", len(ok.Errors))
	}
	if r.cfg.OnSettle != nil {
		r.cfg.OnSettle(Settlement{Batch: batch, Counted: counted, Consolidated: ok.TotalEgress, Errors: ok.Errors})
	}
	if err := r.cfg.Journal.Remove(batch); err != nil {
		return err
	}
	if err := os.Remove(r.trackedPath(batch)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("removing track record: %w", err)
	}
	return nil
}

// count sums the reporter's measure over the batch's records.
func (r *Reporter) count(batch cid.Cid) (uint64, error) {
	f, err := r.cfg.Journal.Open(batch)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var total uint64
	for rec, err := range ReadBatch(f) {
		if err != nil {
			return 0, err
		}
		n, err := r.cfg.Measure(rec)
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

func (r *Reporter) trackedPath(batch cid.Cid) string {
	return r.cfg.Journal.batchPath(batch) + trackSuffix
}

// readTracked returns what is recorded about the batch's track, or nil if the
// tracker has not accepted the batch yet.
func (r *Reporter) readTracked(batch cid.Cid) (*tracked, error) {
	data, err := os.ReadFile(r.trackedPath(batch))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading track record: %w", err)
	}
	ct, err := container.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("decoding track record: %w", err)
	}
	var t tracked
	for _, inv := range ct.Invocations() {
		switch inv.Command().String() {
		case egresscmds.Track.Command.String():
			t.track = inv
		case egresscmds.Consolidate.Command.String():
			t.consolidate = inv
		}
	}
	if t.track == nil || t.consolidate == nil {
		return nil, errors.New("track record is incomplete")
	}
	return &t, nil
}

// writeTracked records the batch's track beside it, replacing the file in one
// step so a crash never leaves half of one.
func (r *Reporter) writeTracked(batch cid.Cid, t *tracked) error {
	data, err := container.Encode(container.Raw, container.New(container.WithInvocations(t.track, t.consolidate)))
	if err != nil {
		return fmt.Errorf("encoding track record: %w", err)
	}
	path := r.trackedPath(batch)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing track record: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("writing track record: %w", err)
	}
	return nil
}

// findConsolidate returns the `/egress/consolidate` invocation caused by the
// track task.
func findConsolidate(invocations []ucan.Invocation, track cid.Cid) (ucan.Invocation, error) {
	for _, inv := range invocations {
		if inv.Command().String() != egresscmds.Consolidate.Command.String() {
			continue
		}
		var args egresscmds.ConsolidateArguments
		if err := args.UnmarshalCBOR(bytes.NewReader(inv.ArgumentsBytes())); err != nil {
			return nil, fmt.Errorf("decoding consolidation arguments: %w", err)
		}
		if args.Cause.Equals(track) {
			return inv, nil
		}
	}
	return nil, errors.New("the response carries no consolidation task")
}

// MeasureContent returns the egress a content record accounts for. A
// `/content/serve` record counts the bytes its receipt says were sent, and a
// `/content/retrieve` record the size of the range its invocation asked for.
func MeasureContent(rec Record) (uint64, error) {
	switch rec.Invocation.Command().String() {
	case content.Serve.Command.String():
		ok, err := content.Serve.Unpack(rec.Receipt)
		if err != nil {
			return 0, err
		}
		return ok.Bytes, nil
	case content.Retrieve.Command.String():
		var args content.RetrieveArguments
		if err := args.UnmarshalCBOR(bytes.NewReader(rec.Invocation.ArgumentsBytes())); err != nil {
			return 0, fmt.Errorf("decoding retrieve arguments: %w", err)
		}
		if args.Range.End < args.Range.Start {
			return 0, fmt.Errorf("retrieve range [%d, %d] ends before it starts", args.Range.Start, args.Range.End)
		}
		return args.Range.End - args.Range.Start + 1, nil
	default:
		return 0, fmt.Errorf("no measure for %s records", rec.Invocation.Command())
	}
}
