package piece

import (
	"encoding/json"
	"errors"

	"github.com/fil-forge/libforge/piece/digest"
	"github.com/fil-forge/libforge/piece/size"
	commcid "github.com/filecoin-project/go-fil-commcid"
	"github.com/ipfs/go-cid"
)

var ErrWrongCodec = errors.New("must be raw codec")

// PieceLink is a Filecoin v2 piece reference. The underlying CID encodes the
// piece commitment plus padding/height as a multihash; the methods on this
// interface expose those parameters.
type PieceLink interface {
	PaddedSize() uint64
	Padding() uint64
	Height() uint8
	DataCommitment() []byte
	// Link returns the v2 piece CID.
	Link() cid.Cid
	// V1Link returns the equivalent v1 piece CID derived from the raw data
	// commitment, suitable for interop with systems that still use v1.
	V1Link() cid.Cid
	json.Marshaler
}

type pieceLink struct{ cid cid.Cid }

func (p pieceLink) DataCommitment() []byte {
	dc, _ := digest.DataCommitment(p.cid.Hash())
	return dc
}

func (p pieceLink) Height() uint8 {
	h, _ := digest.Height(p.cid.Hash())
	return h
}

func (p pieceLink) Link() cid.Cid {
	return p.cid
}

func (p pieceLink) PaddedSize() uint64 {
	return size.HeightToPaddedSize(p.Height())
}

func (p pieceLink) Padding() uint64 {
	pd, _ := digest.Padding(p.cid.Hash())
	return pd
}

func (p pieceLink) V1Link() cid.Cid {
	dc := p.DataCommitment()
	v1, _ := commcid.DataCommitmentV1ToCID(dc)
	return v1
}

func (p pieceLink) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.cid.String())
}

func FromPieceDigest(pd digest.PieceDigest) PieceLink {
	return pieceLink{cid: cid.NewCidV1(cid.Raw, pd.Bytes())}
}

func FromCid(c cid.Cid) (PieceLink, error) {
	if c.Prefix().Codec != cid.Raw {
		return nil, ErrWrongCodec
	}
	pieceDigest, err := digest.NewPieceDigest(c.Hash())
	if err != nil {
		return nil, err
	}
	return FromPieceDigest(pieceDigest), nil
}

func FromV1CidAndSize(v1 cid.Cid, unpaddedDataSize uint64) (PieceLink, error) {
	commitment, err := commcid.CIDToDataCommitmentV1(v1)
	if err != nil {
		return nil, err
	}
	pieceDigest, err := digest.FromCommitmentAndSize(commitment, unpaddedDataSize)
	if err != nil {
		return nil, err
	}
	return FromPieceDigest(pieceDigest), nil
}
