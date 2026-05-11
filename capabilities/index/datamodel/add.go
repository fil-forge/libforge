package datamodel

import "github.com/ipfs/go-cid"

type AddArgumentsModel struct {
	// Index is a link to the Content Archive (CAR) containing the index.
	Index cid.Cid `cborgen:"index" dagjsongen:"index"`
}
