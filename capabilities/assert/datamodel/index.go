package datamodel

import "github.com/ipfs/go-cid"

type IndexArgumentsModel struct {
	Index cid.Cid `cborgen:"index" dagjsongen:"index"`
}
