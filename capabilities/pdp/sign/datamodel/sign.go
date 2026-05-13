// Package datamodel holds the wire models for the /pdp/sign/* capabilities.
//
// The eip712 signing service exposes four operations:
//
//   - /pdp/sign/dataset/create
//   - /pdp/sign/dataset/delete
//   - /pdp/sign/pieces/add
//   - /pdp/sign/pieces/remove/schedule
//
// Each returns the same shape, an [AuthSignatureModel] carrying the
// eip712-signed bytes the caller submits on-chain.
//
// Wire-level conventions:
//
//   - `*big.Int` is encoded as `[]byte` using a sign-prefixed magnitude:
//     first byte is 0x00 (zero or positive) or 0x01 (negative), the
//     remaining bytes are big-endian magnitude. See [BigIntToBytes] and
//     [BigIntFromBytes] in the parent `sign` package for the helpers.
//
//   - `common.Address` (20 bytes) and `common.Hash` (32 bytes) are plain
//     `[]byte`; length validation is left to the caller.
package datamodel

import "github.com/ipfs/go-cid"

// AuthSignatureModel is the eip712 auth signature returned from every
// /pdp/sign/* invocation. Field semantics mirror
// filecoin-services/go/eip712.AuthSignature.
type AuthSignatureModel struct {
	Signature  []byte `cborgen:"signature" dagjsongen:"signature"`
	V          uint8  `cborgen:"v" dagjsongen:"v"`
	R          []byte `cborgen:"r" dagjsongen:"r"`
	S          []byte `cborgen:"s" dagjsongen:"s"`
	SignedData []byte `cborgen:"signedData" dagjsongen:"signedData"`
	Signer     []byte `cborgen:"signer" dagjsongen:"signer"`
}

// MetadataModel is the eip712 metadata bag attached to dataset & piece
// signing requests. Keys carries insertion order so the wire encoding is
// deterministic across implementations.
type MetadataModel struct {
	Keys   []string          `cborgen:"keys" dagjsongen:"keys"`
	Values map[string]string `cborgen:"values" dagjsongen:"values"`
}

// PieceProofsModel wraps the list of `blob/accept` invocation links proving
// the sub-pieces of one piece in a /pdp/sign/pieces/add request. cborgen
// does not natively encode `[][]cid.Cid`, so this single-level wrapper
// turns the outer slice into `[]PieceProofsModel`.
type PieceProofsModel struct {
	Proofs []cid.Cid `cborgen:"proofs" dagjsongen:"proofs"`
}

// DataSetCreateArgumentsModel are the arguments for /pdp/sign/dataset/create.
type DataSetCreateArgumentsModel struct {
	DataSet  []byte        `cborgen:"dataSet" dagjsongen:"dataSet"`
	Payee    []byte        `cborgen:"payee" dagjsongen:"payee"`
	Metadata MetadataModel `cborgen:"metadata" dagjsongen:"metadata"`
}

// DataSetDeleteArgumentsModel are the arguments for /pdp/sign/dataset/delete.
type DataSetDeleteArgumentsModel struct {
	DataSet []byte `cborgen:"dataSet" dagjsongen:"dataSet"`
}

// PiecesAddArgumentsModel are the arguments for /pdp/sign/pieces/add.
//
// Each entry in PieceData has a matching Metadata entry and a matching
// Proofs entry. The Proofs[i].Proofs list links to `blob/accept`
// invocations for every sub-piece of piece i. Each `blob/accept` receipt
// MUST include the corresponding `pdp/accept` effect receipt; all receipt
// data MUST be attached to the signing invocation's container as proof
// blocks.
type PiecesAddArgumentsModel struct {
	DataSet   []byte             `cborgen:"dataSet" dagjsongen:"dataSet"`
	Nonce     []byte             `cborgen:"nonce" dagjsongen:"nonce"`
	PieceData [][]byte           `cborgen:"pieceData" dagjsongen:"pieceData"`
	Metadata  []MetadataModel    `cborgen:"metadata" dagjsongen:"metadata"`
	Proofs    []PieceProofsModel `cborgen:"proofs" dagjsongen:"proofs"`
}

// PiecesRemoveScheduleArgumentsModel are the arguments for
// /pdp/sign/pieces/remove/schedule.
type PiecesRemoveScheduleArgumentsModel struct {
	DataSet []byte   `cborgen:"dataSet" dagjsongen:"dataSet"`
	Pieces  [][]byte `cborgen:"pieces" dagjsongen:"pieces"`
}
