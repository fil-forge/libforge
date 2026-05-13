package blobindex

import (
	"bytes"
	"fmt"
	"io"
	"slices"

	dm "github.com/fil-forge/libforge/blobindex/datamodel"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-car"
	"github.com/ipld/go-car/util"
	mh "github.com/multiformats/go-multihash"
)

// Extract extracts a sharded dag index from a car
func Extract(r io.Reader) (*MapShardedDagIndex, error) {
	dc, err := decodeIndexCar(r)
	if err != nil {
		return nil, NewDecodeFailureError(err.Error())
	}
	return View(dc.root, dc.blocks)
}

// indexCar is the decoded view of an index CAR file.
type indexCar struct {
	root   cid.Cid
	blocks map[cid.Cid][]byte
}

func decodeIndexCar(r io.Reader) (indexCar, error) {
	rd, err := car.NewCarReader(r)
	if err != nil {
		return indexCar{}, fmt.Errorf("creating CAR reader: %w", err)
	}
	if len(rd.Header.Roots) != 1 {
		return indexCar{}, fmt.Errorf("expected exactly one root, got: %d", len(rd.Header.Roots))
	}
	codec := rd.Header.Roots[0].Prefix().Codec
	if codec != cid.DagCBOR {
		return indexCar{}, fmt.Errorf("unexpected root CID codec: %x", codec)
	}
	data := indexCar{root: rd.Header.Roots[0], blocks: map[cid.Cid][]byte{}}
	for {
		blk, err := rd.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return indexCar{}, fmt.Errorf("reading next block: %w", err)
		}
		data.blocks[blk.Cid()] = blk.RawData()
	}
	if _, ok := data.blocks[data.root]; !ok {
		return indexCar{}, fmt.Errorf("missing root block: %s", data.root)
	}
	return data, nil
}

func View(root cid.Cid, blocks map[cid.Cid][]byte) (*MapShardedDagIndex, error) {
	rootBlock, ok := blocks[root]
	if !ok {
		return nil, NewDecodeFailureError(fmt.Sprintf("missing root block: %s", root))
	}

	var shardedDagIndexData dm.ShardedDagIndexModel
	if err := shardedDagIndexData.UnmarshalCBOR(bytes.NewReader(rootBlock)); err != nil {
		return nil, NewDecodeFailureError(fmt.Sprintf("decoding root block: %s: %v", root, err))
	}
	if shardedDagIndexData.DagO_1 == nil {
		return nil, NewUnknownFormatError("unknown index version")
	}

	dagIndex := NewShardedDagIndex(len(shardedDagIndexData.DagO_1.Shards))
	for _, shardLink := range shardedDagIndexData.DagO_1.Shards {
		shard, ok := blocks[shardLink]
		if !ok {
			return nil, NewDecodeFailureError("missing shard block: %s", shardLink)
		}
		var blobIndexData dm.BlobIndexModel
		err := blobIndexData.UnmarshalCBOR(bytes.NewReader(shard))
		if err != nil {
			return nil, NewDecodeFailureError(err.Error())
		}
		blobIndex := NewMultihashMap[Range](len(blobIndexData.Slices))
		for _, blobSlice := range blobIndexData.Slices {
			blobIndex.Set(blobSlice.Digest, blobSlice.Range)
		}
		dagIndex.Shards().Set(blobIndexData.Digest, blobIndex)
	}
	return dagIndex, nil
}

// MapShardedDagIndex is an in-memory implementation of ShardedDagIndex
// using [MultihashMap].
type MapShardedDagIndex struct {
	shards MultihashMap[MultihashMap[Range]]
}

// NewShardedDagIndex constructs an empty sharded DAG index.
// shardSizeHint is used to preallocate the number of shards that will be added.
// Set to -1 if unknown.
func NewShardedDagIndex(shardSizeHint int) *MapShardedDagIndex {
	return &MapShardedDagIndex{NewMultihashMap[MultihashMap[Range]](shardSizeHint)}
}

func (sdi *MapShardedDagIndex) Shards() MultihashMap[MultihashMap[Range]] {
	return sdi.shards
}

func (sdi *MapShardedDagIndex) SetSlice(shard mh.Multihash, slice mh.Multihash, byteRange Range) {
	index := sdi.shards.Get(shard)
	if index == nil {
		index = NewMultihashMap[Range](-1)
		sdi.shards.Set(shard, index)
	}
	index.Set(slice, byteRange)
}

func (sdi *MapShardedDagIndex) Archive() (io.Reader, error) {
	return Archive(sdi)
}

// NewUnknownFormatError returns an error for an unknown format.
func NewUnknownFormatError(reason string, args ...any) error {
	return fmt.Errorf(fmt.Sprintf("unknown format: %s", reason), args...)
}

// NewDecodeFailureError returns an error for a decode failure.
func NewDecodeFailureError(reason string, args ...any) error {
	return fmt.Errorf(fmt.Sprintf("decode failure: %s", reason), args...)
}

// Archive writes a ShardedDagIndex to a CAR file
func Archive(index ShardedDagIndex) (io.Reader, error) {
	// assemble blob index shards
	blobIndexDatas, err := toList(index.Shards(), func(shardHash mh.Multihash, shard MultihashMap[Range]) (dm.BlobIndexModel, error) {
		// assemble blob slices
		blobSliceDatas, err := toList(shard, func(sliceHash mh.Multihash, byteRange Range) (dm.BlobSliceModel, error) {
			return dm.BlobSliceModel{Digest: sliceHash, Range: byteRange}, nil
		})
		if err != nil {
			return dm.BlobIndexModel{}, err
		}
		// sort blob slices
		if err := sortByDigest(blobSliceDatas, func(bsm dm.BlobSliceModel) mh.Multihash {
			return bsm.Digest
		}); err != nil {
			return dm.BlobIndexModel{}, err
		}
		return dm.BlobIndexModel{
			Digest: shardHash,
			Slices: blobSliceDatas,
		}, nil
	})
	if err != nil {
		return nil, err
	}
	// sort blob index shards
	if err := sortByDigest(blobIndexDatas, func(bim dm.BlobIndexModel) mh.Multihash {
		return bim.Digest
	}); err != nil {
		return nil, err
	}

	// initialize root sharded dag index
	shardedDagIndex := dm.ShardedDagIndexModel_0_1{
		Shards: make([]cid.Cid, 0, len(blobIndexDatas)),
	}
	// encode blob index shards to blocks and add links to sharded dag index
	blks := make([]blocks.Block, 0, len(blobIndexDatas)+1)
	for _, shard := range blobIndexDatas {
		var buf bytes.Buffer
		err := shard.MarshalCBOR(&buf)
		if err != nil {
			return nil, err
		}
		b := buf.Bytes()
		l, err := cid.V1Builder{Codec: cid.DagCBOR, MhType: mh.SHA2_256}.Sum(b)
		if err != nil {
			return nil, err
		}
		blk, err := blocks.NewBlockWithCid(b, l)
		if err != nil {
			return nil, err
		}
		blks = append(blks, blk)
		shardedDagIndex.Shards = append(shardedDagIndex.Shards, l)
	}

	// encode the root block
	model := dm.ShardedDagIndexModel{DagO_1: &shardedDagIndex}
	var rootData bytes.Buffer
	if err := model.MarshalCBOR(&rootData); err != nil {
		return nil, err
	}
	root, err := cid.V1Builder{Codec: cid.DagCBOR, MhType: mh.SHA2_256}.Sum(rootData.Bytes())
	if err != nil {
		return nil, err
	}
	rootBlock, err := blocks.NewBlockWithCid(rootData.Bytes(), root)
	if err != nil {
		return nil, err
	}

	reader, writer := io.Pipe()
	go func() {
		err := car.WriteHeader(&car.CarHeader{Roots: []cid.Cid{root}, Version: 1}, writer)
		if err != nil {
			writer.CloseWithError(fmt.Errorf("writing CAR header: %w", err))
			return
		}
		for _, block := range append(blks, rootBlock) {
			err = util.LdWrite(writer, block.Cid().Bytes(), block.RawData())
			if err != nil {
				writer.CloseWithError(fmt.Errorf("writing CAR blocks: %w", err))
				return
			}
		}
		writer.Close()
	}()
	return reader, nil
}

func toList[E, T any](mhm MultihashMap[T], newElem func(mh.Multihash, T) (E, error)) ([]E, error) {
	asList := make([]E, 0, mhm.Size())
	for hash, value := range mhm.Iterator() {
		e, err := newElem(hash, value)
		if err != nil {
			return nil, err
		}
		asList = append(asList, e)
	}
	return asList, nil
}

func sortByDigest[E any](list []E, getDigest func(E) mh.Multihash) error {
	decodeds := NewMultihashMap[*mh.DecodedMultihash](len(list))
	for _, e := range list {
		hash := getDigest(e)
		decoded, err := mh.Decode(hash)
		if err != nil {
			return err
		}
		decodeds.Set(hash, decoded)
	}
	slices.SortFunc(list, func(a, b E) int {
		decodedA := decodeds.Get(getDigest(a))
		decodedB := decodeds.Get(getDigest(b))
		return bytes.Compare(decodedA.Digest, decodedB.Digest)
	})
	return nil
}
