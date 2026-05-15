package datamodel

import (
	"github.com/fil-forge/libforge/capabilities"
	cid "github.com/ipfs/go-cid"
)

type AllocateArgumentsModel struct {
	Blob  BlobModel `cborgen:"blob" dagjsongen:"blob"`
	Cause cid.Cid   `cborgen:"cause" dagjsongen:"cause"`
}

type AllocateOKModel struct {
	Size    int64             `cborgen:"size" dagjsongen:"size"`
	Address *BlobAddressModel `cborgen:"address,omitempty" dagjsongen:"address,omitempty"`
}

type BlobAddressModel struct {
	URL     capabilities.CborURL `cborgen:"url" dagjsongen:"url"`
	Headers map[string]string    `cborgen:"headers" dagjsongen:"headers"`
	Expires int64                `cborgen:"expires" dagjsongen:"expires"`
}
