package datamodel

import (
	"github.com/fil-forge/ucantone/ucan/promise"
	"github.com/multiformats/go-multihash"
)

type BlobModel struct {
	Digest multihash.Multihash `cborgen:"digest" dagjsongen:"digest"`
	Size   int64               `cborgen:"size" dagjsongen:"size"`
}

type AddArgumentsModel struct {
	Blob BlobModel `cborgen:"blob" dagjsongen:"blob"`
}

type AddOKModel struct {
	// Site is a promise of the `/blob/accept` task result, which contains a
	// location claim for the blob, describing where it can be retrieved from.
	Site promise.AwaitOK `cborgen:"site" dagjsongen:"site"`
}
