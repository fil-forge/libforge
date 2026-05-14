package datamodel

import (
	"github.com/fil-forge/ucantone/ucan/promise"
	cid "github.com/ipfs/go-cid"
)

type AcceptArgumentsModel struct {
	Blob BlobModel       `cborgen:"blob" dagjsongen:"blob"`
	Put  promise.AwaitOK `cborgen:"_put" dagjsongen:"_put"`
}

type AcceptOKModel struct {
	Site cid.Cid `cborgen:"site" dagjsongen:"site"`
}
