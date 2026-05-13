package blobindex_test

import (
	"io"
	"math/rand/v2"
	"testing"

	"github.com/fil-forge/libforge/blobindex"
	"github.com/fil-forge/libforge/testutil"
	"github.com/ipfs/go-cid"
	multihash "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
)

func TestFromToArchive(t *testing.T) {
	testIndexData := map[cid.Cid][]struct {
		digest    multihash.Multihash
		byteRange []int
	}{
		testutil.RandomCID(t): {
			{digest: testutil.RandomMultihash(t), byteRange: []int{0, 99}},
			{digest: testutil.RandomMultihash(t), byteRange: []int{100, 199}},
			{digest: testutil.RandomMultihash(t), byteRange: []int{200, 299}},
		},
		testutil.RandomCID(t): {
			{digest: testutil.RandomMultihash(t), byteRange: []int{0, 34}},
			{digest: testutil.RandomMultihash(t), byteRange: []int{35, 58}},
		},
	}

	index := blobindex.NewShardedDagIndex(len(testIndexData))
	for shard, slices := range testIndexData {
		for _, slice := range slices {
			index.SetSlice(
				shard.Hash(),
				slice.digest,
				blobindex.Range{
					Start: int64(slice.byteRange[0]),
					End:   int64(slice.byteRange[1]),
				},
			)
		}
	}

	// ensure all the data is in the index
	require.Equal(t, len(testIndexData), index.Shards().Size())
	for shard, slices := range testIndexData {
		require.True(t, index.Shards().Has(shard.Hash()))

		idxSlices := index.Shards().Get(shard.Hash())
		require.Equal(t, len(slices), idxSlices.Size())

		for _, slice := range slices {
			require.True(t, idxSlices.Has(slice.digest))
			byteRange := idxSlices.Get(slice.digest)
			require.Equal(t, int64(slice.byteRange[0]), byteRange.Start)
			require.Equal(t, int64(slice.byteRange[1]), byteRange.End)
		}
	}

	t.Run("round trip", func(t *testing.T) {
		r, err := index.Archive()
		require.NoError(t, err)

		newIndex, err := blobindex.Extract(r)
		require.NoError(t, err)

		// ensure all the data is in the new index
		require.Equal(t, index.Shards().Size(), newIndex.Shards().Size())
		for shard, slices := range index.Shards().Iterator() {
			require.True(t, newIndex.Shards().Has(shard))

			newSlices := newIndex.Shards().Get(shard)
			require.Equal(t, slices.Size(), newSlices.Size())

			for slice, byteRange := range slices.Iterator() {
				require.True(t, newSlices.Has(slice))
				newByteRange := newSlices.Get(slice)
				require.Equal(t, byteRange.Start, newByteRange.Start)
				require.Equal(t, byteRange.End, newByteRange.End)
			}
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		index1 := blobindex.NewShardedDagIndex(len(testIndexData))
		for shard, slices := range testIndexData {
			// shuffle the order slices are added
			rand.Shuffle(len(slices), func(i, j int) {
				slices[i], slices[j] = slices[j], slices[i]
			})
			for _, slice := range slices {
				index1.SetSlice(
					shard.Hash(),
					slice.digest,
					blobindex.Range{
						Start: int64(slice.byteRange[0]),
						End:   int64(slice.byteRange[1]),
					},
				)
			}
		}

		idxArchive := testutil.Must(io.ReadAll(testutil.Must(index.Archive())(t)))(t)
		idx1Archive := testutil.Must(io.ReadAll(testutil.Must(index1.Archive())(t)))(t)
		require.Equal(t, idxArchive, idx1Archive)
	})
}
