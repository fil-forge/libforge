package datamodel

import (
	"github.com/multiformats/go-multihash"
)

// RetrieveBlobModel identifies a blob solely by its content multihash. Used
// by the service-level `/blob/retrieve` capability where the caller is
// fetching data by hash without prior knowledge of the byte size.
type RetrieveBlobModel struct {
	Digest multihash.Multihash `cborgen:"digest" dagjsongen:"digest"`
}

// RetrieveArgumentsModel is the argument shape of the `/blob/retrieve`
// capability — a service-level (not space-scoped) retrieval handle. Compare
// to `content.RetrieveArguments` which is space-scoped and carries a byte
// Range; the `/blob/retrieve` flow is consumed by service principals (e.g.
// the indexer) fetching content claims that aren't bound to any space.
type RetrieveArgumentsModel struct {
	Blob RetrieveBlobModel `cborgen:"blob" dagjsongen:"blob"`
}

// RetrieveOKModel is the success return for `/blob/retrieve`. The blob bytes
// themselves are streamed back through the response container's body (the
// libforge HTTPHeader retrieval transport); the typed OK record is empty.
type RetrieveOKModel struct{}
