package datamodel

import (
	"github.com/multiformats/go-multihash"
)

type BlobModel struct {
	Digest multihash.Multihash `cborgen:"digest" dagjsongen:"digest"`
}

type RangeModel struct {
	Start int64 `cborgen:"start" dagjsongen:"start"`
	End   int64 `cborgen:"end" dagjsongen:"end"`
}

type RetrieveArgumentsModel struct {
	Blob  BlobModel  `cborgen:"blob" dagjsongen:"blob"`
	Range RangeModel `cborgen:"range" dagjsongen:"range"`
}

type RetrieveOKModel struct{}
