// Package egress journals the receipts of served content, seals them into
// batches and reports the batches to an egress tracking service.
//
// A host records each retrieval it serves with [Journal.Append]. The journal
// seals a batch when it is large or old enough, a [Reporter] submits each
// sealed batch to the tracker with `/egress/track` and removes it once the
// tracker has consolidated it, and the handler from [NewHandler] serves the
// batches for the tracker to fetch.
package egress

import (
	"errors"
	"fmt"
	"io"
	"iter"

	"github.com/fil-forge/automobile"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/ucan/delegation"
	"github.com/fil-forge/ucantone/ucan/invocation"
	"github.com/fil-forge/ucantone/ucan/receipt"
	"github.com/ipfs/go-cid"
)

// Record is one served retrieval: the receipt for it, the invocation whose
// task the receipt is for, and the delegations that authorized the invocation.
type Record struct {
	// Receipt attests that the task ran.
	Receipt ucan.Receipt
	// Invocation is the invocation whose task the receipt is for.
	Invocation ucan.Invocation
	// Proofs are the delegations the invocation names as its proofs.
	Proofs []ucan.Delegation
}

// validate checks the record is complete: the receipt is for the invocation's
// task and every proof the invocation names is present.
func (r Record) validate() error {
	if r.Receipt == nil {
		return errors.New("record has no receipt")
	}
	if r.Invocation == nil {
		return errors.New("record has no invocation")
	}
	if task := r.Invocation.Task().Link(); !r.Receipt.Ran().Equals(task) {
		return fmt.Errorf("receipt is for task %s, the invocation's task is %s", r.Receipt.Ran(), task)
	}
	for _, link := range r.Invocation.Proofs() {
		found := false
		for _, p := range r.Proofs {
			if p.Link().Equals(link) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("record is missing proof %s", link)
		}
	}
	return nil
}

// ReadBatch reads the records of a batch.
//
// A batch is a CAR file with no roots in which each block is a UCAN token. A
// record's blocks are written in order: the delegations the batch does not
// already hold, then the invocation, then the receipt. A delegation appears
// once however many records refer to it, so a batch is read front to back
// with only its delegations held in memory.
//
// A block that is not a token, a receipt with no invocation before it and an
// invocation whose proofs are missing are each reported as an error for that
// record, and reading continues. An error reading the CAR itself ends the
// sequence.
func ReadBatch(r io.Reader) iter.Seq2[Record, error] {
	return func(yield func(Record, error) bool) {
		cr, err := automobile.NewReader(r)
		if err != nil {
			yield(Record{}, fmt.Errorf("reading batch header: %w", err))
			return
		}
		delegations := map[cid.Cid]ucan.Delegation{}
		// Invocations waiting for their receipt, by task.
		pending := map[cid.Cid]ucan.Invocation{}
		for {
			blk, err := cr.Read()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				yield(Record{}, fmt.Errorf("reading batch block: %w", err))
				return
			}
			// The same order ucantone's container decodes in: a receipt is an
			// invocation on the wire, so it has to be tried first.
			if dlg, err := delegation.Decode(blk.Data); err == nil {
				delegations[blk.Link] = dlg
				continue
			}
			if rcpt, err := receipt.Decode(blk.Data); err == nil {
				inv, ok := pending[rcpt.Ran()]
				if !ok {
					if !yield(Record{}, fmt.Errorf("receipt %s: no invocation for task %s", blk.Link, rcpt.Ran())) {
						return
					}
					continue
				}
				delete(pending, rcpt.Ran())
				rec := Record{Receipt: rcpt, Invocation: inv}
				var missing error
				for _, link := range inv.Proofs() {
					dlg, ok := delegations[link]
					if !ok {
						missing = fmt.Errorf("receipt %s: missing proof %s", blk.Link, link)
						break
					}
					rec.Proofs = append(rec.Proofs, dlg)
				}
				if missing != nil {
					rec = Record{}
				}
				if !yield(rec, missing) {
					return
				}
				continue
			}
			if inv, err := invocation.Decode(blk.Data); err == nil {
				pending[inv.Task().Link()] = inv
				continue
			}
			if !yield(Record{}, fmt.Errorf("block %s is not a UCAN token", blk.Link)) {
				return
			}
		}
	}
}

// recordBlocks returns the blocks of a record in the order a batch holds
// them, leaving out the delegations for which have reports true.
func recordBlocks(rec Record, have func(cid.Cid) bool) []automobile.Block {
	blocks := make([]automobile.Block, 0, len(rec.Proofs)+2)
	for _, p := range rec.Proofs {
		if have(p.Link()) {
			continue
		}
		blocks = append(blocks, automobile.Block{Link: p.Link(), Data: p.Bytes()})
	}
	blocks = append(blocks,
		automobile.Block{Link: rec.Invocation.Link(), Data: rec.Invocation.Bytes()},
		automobile.Block{Link: rec.Receipt.Link(), Data: rec.Receipt.Bytes()},
	)
	return blocks
}
