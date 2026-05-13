package datamodel

import (
	"github.com/fil-forge/libforge/merkletree"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

type InfoArgumentsModel struct {
	Blob multihash.Multihash `cborgen:"blob" dagjsongen:"blob"`
}

type InfoAcceptedAggregateModel struct {
	Aggregate      cid.Cid             `cborgen:"aggregate" dagjsongen:"aggregate"`
	InclusionProof merkletree.ProofData `cborgen:"inclusionProof" dagjsongen:"inclusionProof"`
}

type InfoOKModel struct {
	Piece      cid.Cid                      `cborgen:"piece" dagjsongen:"piece"`
	Aggregates []InfoAcceptedAggregateModel `cborgen:"aggregates" dagjsongen:"aggregates"`
}
