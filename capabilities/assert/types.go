package assert

import (
	"github.com/fil-forge/libforge/capabilities"
	"github.com/fil-forge/ucantone/did"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

type IndexArguments struct {
	Index cid.Cid `cborgen:"index" dagjsongen:"index"`
}

type LocationArguments struct {
	Space    did.DID                `cborgen:"space" dagjsongen:"space"`
	Content  multihash.Multihash    `cborgen:"content" dagjsongen:"content"`
	Location []capabilities.CborURL `cborgen:"location" dagjsongen:"location"`
	Range    *Range                 `cborgen:"range,omitempty" dagjsongen:"range,omitempty"`
}

type Range struct {
	Offset uint64  `cborgen:"offset" dagjsongen:"offset"`
	Length *uint64 `cborgen:"length,omitempty" dagjsongen:"length,omitempty"`
}

type EqualsArguments struct {
	Content multihash.Multihash `cborgen:"content" dagjsongen:"content"`
	Equals  cid.Cid             `cborgen:"equals" dagjsongen:"equals"`
}
