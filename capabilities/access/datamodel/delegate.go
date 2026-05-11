package datamodel

import "github.com/ipfs/go-cid"

type DelegateArgumentsModel struct {
	// The delegations to store.
	Delegations []cid.Cid `cborgen:"delegations" dagjsongen:"delegations"`
}
