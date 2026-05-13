package datamodel

import (
	"github.com/fil-forge/libforge/merkletree"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

type AcceptArgumentsModel struct {
	Blob multihash.Multihash `cborgen:"blob" dagjsongen:"blob"`
}

type AcceptOKModel struct {
	Aggregate      cid.Cid             `cborgen:"aggregate" dagjsongen:"aggregate"`
	InclusionProof merkletree.ProofData `cborgen:"inclusionProof" dagjsongen:"inclusionProof"`
	Piece          cid.Cid             `cborgen:"piece" dagjsongen:"piece"`
}
