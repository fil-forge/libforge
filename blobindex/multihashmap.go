package blobindex

import (
	"github.com/fil-forge/libforge/bytemap"
	mh "github.com/multiformats/go-multihash"
)

// NewMultihashMap returns a new map of multihash to a data type
func NewMultihashMap[T any](sizeHint int) MultihashMap[T] {
	return bytemap.NewByteMap[mh.Multihash, T](sizeHint)
}
