package datamodel

import (
	"github.com/fil-forge/libforge/capabilities"
	"github.com/fil-forge/ucantone/did"
	"github.com/multiformats/go-multihash"
)

type LocationArgumentsModel struct {
	Space    did.DID                `cborgen:"space" dagjsongen:"space"`
	Content  multihash.Multihash    `cborgen:"content" dagjsongen:"content"`
	Location []capabilities.CborURL `cborgen:"location" dagjsongen:"location"`
	Range    *RangeModel            `cborgen:"range,omitempty" dagjsongen:"range,omitempty"`
}

type RangeModel struct {
	Start int64  `cborgen:"start" dagjsongen:"start"`
	End   *int64 `cborgen:"end,omitempty" dagjsongen:"end,omitempty"`
}
