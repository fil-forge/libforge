package datamodel

import cid "github.com/ipfs/go-cid"

type ClaimOKModel struct {
	// Delegations is a list of the CIDs of delegations granted for the request.
	Delegations []cid.Cid `cborgen:"delegations" dagjsongen:"delegations"`
}
