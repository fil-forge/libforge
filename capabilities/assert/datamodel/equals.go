package datamodel

import (
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

type EqualsArgumentsModel struct {
	Content multihash.Multihash `cborgen:"content" dagjsongen:"content"`
	Equals  cid.Cid             `cborgen:"equals" dagjsongen:"equals"`
}
