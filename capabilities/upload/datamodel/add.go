package datamodel

import (
	"github.com/ipfs/go-cid"
)

type AddArgumentsModel struct {
	Root   cid.Cid   `cborgen:"root" dagjsongen:"root"`
	Shards []cid.Cid `cborgen:"shards" dagjsongen:"shards"`
	Index  *cid.Cid  `cborgen:"index,omitempty" dagjsongen:"index,omitempty"`
}
