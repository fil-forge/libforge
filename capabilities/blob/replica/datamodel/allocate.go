package datamodel

import (
	"github.com/fil-forge/ucantone/ucan/promise"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

type BlobModel struct {
	Digest multihash.Multihash `cborgen:"digest"`
	Size   int64               `cborgen:"size"`
}

type AllocateArgumentsModel struct {
	// Blob is the blob to be allocated.
	Blob BlobModel `cborgen:"blob"`
	// Site is a link to a location commitment indicating where the Blob must be
	// fetched from.
	Site cid.Cid `cborgen:"site"`
	// Cause is a link to the `/blob/replicate` task that caused this allocation.
	Cause cid.Cid `cborgen:"cause"`
}

type AllocateOKModel struct {
	// Site resolves to an additional location for the blob.
	// It is a link to a /blob/replica/transfer task.
	Site promise.AwaitOK `cborgen:"site"`
}
