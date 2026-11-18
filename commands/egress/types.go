package egress

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/ipfs/go-cid"
)

// TrackArguments is the argument shape of `/egress/track`.
type TrackArguments struct {
	// Receipts is the CID of the batch of receipts the reporter has staged.
	Receipts cid.Cid `cborgen:"receipts" dagjsongen:"receipts"`
	// Endpoint is the URL template the batch can be fetched from, with a
	// `{cid}` placeholder for the batch CID.
	Endpoint commands.CborURL `cborgen:"endpoint" dagjsongen:"endpoint"`
}

// TrackOK is the success return for `/egress/track`. The response also carries
// the `/egress/consolidate` invocation that will process the batch.
type TrackOK struct{}

// ConsolidateArguments is the argument shape of `/egress/consolidate`.
type ConsolidateArguments struct {
	// Cause is the task link of the `/egress/track` invocation that submitted
	// the batch.
	Cause cid.Cid `cborgen:"cause" dagjsongen:"cause"`
}

// ConsolidateOK is the success return for `/egress/consolidate`.
type ConsolidateOK struct {
	// TotalEgress is the bytes of egress accounted for by the receipts that
	// were processed.
	TotalEgress uint64 `cborgen:"totalEgress" dagjsongen:"totalEgress"`
	// Errors lists the receipts that could not be processed. It is empty when
	// every receipt in the batch was.
	Errors []ReceiptError `cborgen:"errors" dagjsongen:"errors"`
}

// ReceiptError describes a receipt in a batch that could not be processed.
type ReceiptError struct {
	// Name is the machine readable name of the error.
	Name string `cborgen:"name" dagjsongen:"name"`
	// Message is a human readable description of the error.
	Message string `cborgen:"message" dagjsongen:"message"`
	// Receipt is the link of the receipt that failed.
	Receipt cid.Cid `cborgen:"receipt" dagjsongen:"receipt"`
}
