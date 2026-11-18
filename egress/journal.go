package egress

import (
	"bufio"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fil-forge/automobile"
	"github.com/fil-forge/automobile/datamodel"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multihash"
)

var log = logging.Logger("egress")

const (
	// DefaultMaxBatchSize is the size at which a batch is sealed.
	DefaultMaxBatchSize int64 = 100 << 20
	// MaxBatchSize is the largest batch the egress tracking service accepts.
	MaxBatchSize int64 = 1 << 30
	// DefaultMaxBatchAge is how long a batch stays open before it is sealed,
	// counted from its first record.
	DefaultMaxBatchAge = time.Hour
	// DefaultFlushInterval is how often buffered records are written to disk.
	// A crash loses at most this much.
	DefaultFlushInterval = time.Second
)

const (
	openName    = "egress.open"
	batchSuffix = ".car"
)

// ErrNotFound is returned for a batch the journal does not hold.
var ErrNotFound = errors.New("batch not found")

type journalConfig struct {
	maxSize    int64
	maxAge     time.Duration
	flushEvery time.Duration
	now        func() time.Time
}

// JournalOption configures a [Journal].
type JournalOption func(*journalConfig)

// WithMaxBatchSize sets the most bytes a batch may hold: a batch is sealed
// before a record that would take it past this. The default is
// [DefaultMaxBatchSize] and the most is [MaxBatchSize]. A single record
// larger than the limit gets a batch to itself.
func WithMaxBatchSize(n int64) JournalOption {
	return func(c *journalConfig) { c.maxSize = n }
}

// WithMaxBatchAge sets how long a batch stays open, counted from its first
// record, before it is sealed whatever its size. The default is
// [DefaultMaxBatchAge].
func WithMaxBatchAge(d time.Duration) JournalOption {
	return func(c *journalConfig) { c.maxAge = d }
}

// WithFlushInterval sets how often buffered records are written to disk. The
// default is [DefaultFlushInterval].
func WithFlushInterval(d time.Duration) JournalOption {
	return func(c *journalConfig) { c.flushEvery = d }
}

// Journal appends records to an open batch in a directory and seals the batch
// when it reaches its size or age limit. A sealed batch is a file named by
// the CID of its bytes and never changes.
//
// At most one Journal may use a directory at a time.
type Journal struct {
	dir string
	cfg journalConfig

	mu     sync.Mutex
	closed bool
	// The open batch. file is nil until the first record after a seal.
	file *os.File
	buf  *bufio.Writer
	hash hash.Hash
	size int64
	// records is how many records the open batch holds.
	records int
	opened  time.Time
	// have is the delegations the open batch already holds.
	have map[cid.Cid]struct{}

	sealed chan struct{}
	stop   chan struct{}
	done   chan struct{}
}

// OpenJournal opens the journal in dir, creating the directory if needed. A
// batch left open by an earlier process is sealed: records cut short by a
// crash are dropped from its end first.
func OpenJournal(dir string, options ...JournalOption) (*Journal, error) {
	cfg := journalConfig{
		maxSize:    DefaultMaxBatchSize,
		maxAge:     DefaultMaxBatchAge,
		flushEvery: DefaultFlushInterval,
		now:        time.Now,
	}
	for _, o := range options {
		o(&cfg)
	}
	if cfg.maxSize <= 0 || cfg.maxSize > MaxBatchSize {
		return nil, fmt.Errorf("max batch size %d: must be between 1 and %d bytes", cfg.maxSize, MaxBatchSize)
	}
	if cfg.maxAge <= 0 {
		return nil, fmt.Errorf("max batch age %s: must be positive", cfg.maxAge)
	}
	if cfg.flushEvery <= 0 {
		return nil, fmt.Errorf("flush interval %s: must be positive", cfg.flushEvery)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating journal directory: %w", err)
	}
	j := &Journal{
		dir:    dir,
		cfg:    cfg,
		sealed: make(chan struct{}, 1),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	if err := j.recoverOpen(); err != nil {
		return nil, err
	}
	go j.run()
	return j, nil
}

// Append adds a record to the open batch, sealing the batch first if the
// record would take it past its size limit. A record whose receipt reports a
// failure is dropped: a failed retrieval served nothing.
//
// A write that fails costs the records still buffered. The journal sets the
// open batch aside and the next Append seals what reached the disk and starts
// a new one, so a full disk stops appends only while it is full.
func (j *Journal) Append(ctx context.Context, rec Record) error {
	if err := rec.validate(); err != nil {
		return err
	}
	if rec.Receipt.Out().IsErr() {
		return nil
	}

	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return errors.New("journal is closed")
	}
	if j.file != nil && j.records > 0 && j.size+blocksSize(j.blocks(rec)) > j.cfg.maxSize {
		if _, err := j.seal(); err != nil {
			return fmt.Errorf("sealing batch: %w", err)
		}
	}
	if j.file == nil {
		if err := j.open(); err != nil {
			return err
		}
	}
	// Worked out again after a seal: a new batch holds none of the proofs.
	blocks := j.blocks(rec)
	w := io.MultiWriter(j.buf, j.hash)
	for _, b := range blocks {
		if err := automobile.WriteBlock(w, b); err != nil {
			j.abandon()
			return fmt.Errorf("writing record: %w", err)
		}
	}
	j.size += blocksSize(blocks)
	j.records++
	for _, p := range rec.Proofs {
		j.have[p.Link()] = struct{}{}
	}
	return nil
}

// blocks returns the blocks rec adds to the open batch, or to a new batch if
// none is open. The caller holds the lock.
func (j *Journal) blocks(rec Record) []automobile.Block {
	return recordBlocks(rec, func(link cid.Cid) bool {
		_, ok := j.have[link]
		return ok
	})
}

func blocksSize(blocks []automobile.Block) int64 {
	var n int64
	for _, b := range blocks {
		n += int64(automobile.LdSize(b.Link.Bytes(), b.Data))
	}
	return n
}

// Seal seals the open batch now, whatever its size or age, and returns its
// CID. It reports false if no record has been appended since the last seal.
func (j *Journal) Seal() (cid.Cid, bool, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return cid.Undef, false, nil
	}
	c, err := j.seal()
	if err != nil {
		return cid.Undef, false, err
	}
	return c, true, nil
}

// Sealed returns a channel that receives after a batch is sealed. One receive
// may stand for several seals.
func (j *Journal) Sealed() <-chan struct{} {
	return j.sealed
}

// Batches lists the sealed batches.
func (j *Journal) Batches() ([]cid.Cid, error) {
	entries, err := os.ReadDir(j.dir)
	if err != nil {
		return nil, fmt.Errorf("listing batches: %w", err)
	}
	var batches []cid.Cid
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, batchSuffix) {
			continue
		}
		c, err := cid.Parse(strings.TrimSuffix(name, batchSuffix))
		if err != nil {
			continue
		}
		batches = append(batches, c)
	}
	return batches, nil
}

// Open opens a sealed batch for reading. It returns [ErrNotFound] if the
// journal does not hold the batch.
func (j *Journal) Open(batch cid.Cid) (*os.File, error) {
	f, err := os.Open(j.batchPath(batch))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

// Remove deletes a sealed batch. Removing a batch the journal does not hold
// is not an error.
func (j *Journal) Remove(batch cid.Cid) error {
	err := os.Remove(j.batchPath(batch))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("removing batch: %w", err)
	}
	return nil
}

// Close writes buffered records to disk and stops the journal. The open batch
// stays open, and is sealed when the directory is next opened.
func (j *Journal) Close() error {
	j.mu.Lock()
	if j.closed {
		j.mu.Unlock()
		return nil
	}
	j.closed = true
	var err error
	if j.file != nil {
		err = errors.Join(j.buf.Flush(), j.file.Sync(), j.file.Close())
		j.file = nil
	}
	j.mu.Unlock()
	close(j.stop)
	<-j.done
	return err
}

func (j *Journal) batchPath(batch cid.Cid) string {
	return filepath.Join(j.dir, batch.String()+batchSuffix)
}

func (j *Journal) openPath() string {
	return filepath.Join(j.dir, openName)
}

// run flushes the open batch at each interval and seals it once it is old
// enough.
func (j *Journal) run() {
	defer close(j.done)
	ticker := time.NewTicker(j.cfg.flushEvery)
	defer ticker.Stop()
	for {
		select {
		case <-j.stop:
			return
		case <-ticker.C:
			if err := j.tick(); err != nil {
				log.Errorw("flushing the open batch", "error", err)
			}
		}
	}
}

func (j *Journal) tick() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed || j.file == nil {
		return nil
	}
	if j.cfg.now().Sub(j.opened) >= j.cfg.maxAge {
		_, err := j.seal()
		return err
	}
	if err := j.buf.Flush(); err != nil {
		j.abandon()
		return err
	}
	return nil
}

// abandon gives up on the open batch after a failed write, leaving its file
// for the next open to recover. A writer that has failed once fails every
// write after it, so the batch cannot be carried on with. The caller holds
// the lock.
func (j *Journal) abandon() {
	if j.file != nil {
		_ = j.file.Close()
	}
	j.file, j.buf, j.hash, j.have = nil, nil, nil, nil
	j.size, j.records = 0, 0
}

// open starts a new batch, first sealing a batch that an earlier failure left
// on disk. The caller holds the lock.
func (j *Journal) open() error {
	if err := j.recoverOpen(); err != nil {
		return err
	}
	f, err := os.OpenFile(j.openPath(), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("creating batch: %w", err)
	}
	j.file = f
	j.buf = bufio.NewWriter(f)
	j.hash = sha256.New()
	j.have = map[cid.Cid]struct{}{}
	j.opened = j.cfg.now()
	j.size, j.records = 0, 0
	if err := automobile.WriteHeader(io.MultiWriter(j.buf, j.hash), nil); err != nil {
		j.abandon()
		return fmt.Errorf("writing batch header: %w", err)
	}
	hs, err := automobile.HeaderSize(datamodel.HeaderModel{Version: 1})
	if err != nil {
		j.abandon()
		return err
	}
	j.size = int64(hs)
	return nil
}

// seal closes the open batch and renames it to its CID. The caller holds the
// lock, and the batch is open. Whether or not it succeeds, no batch is open
// afterwards: a batch it could not seal stays on disk for the next open to
// recover.
func (j *Journal) seal() (cid.Cid, error) {
	err := errors.Join(j.buf.Flush(), j.file.Sync(), j.file.Close())
	digest := j.hash.Sum(nil)
	j.file = nil
	j.abandon()
	if err != nil {
		return cid.Undef, fmt.Errorf("closing batch: %w", err)
	}
	batch, err := batchCID(digest)
	if err != nil {
		return cid.Undef, err
	}
	if err := os.Rename(j.openPath(), j.batchPath(batch)); err != nil {
		return cid.Undef, fmt.Errorf("naming batch: %w", err)
	}
	select {
	case j.sealed <- struct{}{}:
	default:
	}
	return batch, nil
}

// recoverOpen seals a batch left on disk unsealed, by an earlier process or
// by a write that failed in this one. Whatever follows the last whole block
// is dropped, and a batch with no blocks is deleted. The caller holds the
// lock, or is the only user of the journal, and no batch is open.
func (j *Journal) recoverOpen() error {
	path := j.openPath()
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("opening the batch left open: %w", err)
	}
	defer f.Close()

	var good int64
	blocks := 0
	if cr, err := automobile.NewReader(f); err == nil {
		hs, err := automobile.HeaderSize(cr.Header)
		if err != nil {
			return err
		}
		good = int64(hs)
		for {
			blk, err := cr.Read()
			if err != nil {
				break
			}
			good += int64(automobile.LdSize(blk.Link.Bytes(), blk.Data))
			blocks++
		}
	}
	if blocks == 0 {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing the empty batch left open: %w", err)
		}
		return nil
	}
	if err := f.Truncate(good); err != nil {
		return fmt.Errorf("trimming the batch left open: %w", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing the batch left open: %w", err)
	}
	if err := f.Sync(); err != nil {
		return err
	}
	batch, err := batchCID(h.Sum(nil))
	if err != nil {
		return err
	}
	if err := os.Rename(path, j.batchPath(batch)); err != nil {
		return fmt.Errorf("naming the batch left open: %w", err)
	}
	select {
	case j.sealed <- struct{}{}:
	default:
	}
	return nil
}

// batchCID is the CID of a batch whose bytes have the given SHA-256 digest.
func batchCID(digest []byte) (cid.Cid, error) {
	mh, err := multihash.Encode(digest, multihash.SHA2_256)
	if err != nil {
		return cid.Undef, fmt.Errorf("encoding batch digest: %w", err)
	}
	return cid.NewCidV1(automobile.Code, mh), nil
}
