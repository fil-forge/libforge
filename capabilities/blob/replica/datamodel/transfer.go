package datamodel

import (
	"github.com/fil-forge/ucantone/ucan/promise"
	"github.com/ipfs/go-cid"
)

type TransferArgumentsModel struct {
	// Blob is the blob to be transferred.
	Blob BlobModel `cborgen:"blob"`
	// Site is a link to a location commitment indicating where the Blob must be
	// fetched from.
	Site cid.Cid `cborgen:"site"`
	// Cause links to the `/blob/replica/allocate` task that initiated this transfer.
	Cause cid.Cid `cborgen:"cause"`
}

type TransferOKModel struct {
	// Site links to the location commitment that indicate where the Blob has been
	// transferred to.
	Site cid.Cid `cborgen:"site"`
	// PDP links to the /pdp/accept task that will resolve when aggregation
	// is complete and the piece is accepted.
	PDP promise.AwaitOK `cborgen:"pdp"`
}
