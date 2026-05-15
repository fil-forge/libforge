package blob

import (
	"github.com/fil-forge/libforge/capabilities"
	"github.com/fil-forge/ucantone/ucan/promise"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
)

type Blob struct {
	Digest multihash.Multihash `cborgen:"digest" dagjsongen:"digest"`
	Size   uint64              `cborgen:"size" dagjsongen:"size"`
}

type AddArguments struct {
	Blob Blob `cborgen:"blob" dagjsongen:"blob"`
}

type AddOK struct {
	// Site is a promise of the `/blob/accept` task result, which contains a
	// location claim for the blob, describing where it can be retrieved from.
	Site promise.AwaitOK `cborgen:"site" dagjsongen:"site"`
}

type AcceptArguments struct {
	Blob Blob            `cborgen:"blob" dagjsongen:"blob"`
	Put  promise.AwaitOK `cborgen:"_put" dagjsongen:"_put"`
}

type AcceptOK struct {
	Site cid.Cid `cborgen:"site" dagjsongen:"site"`
}

type AllocateArguments struct {
	Blob  Blob    `cborgen:"blob" dagjsongen:"blob"`
	Cause cid.Cid `cborgen:"cause" dagjsongen:"cause"`
}

type AllocateOK struct {
	Size    uint64       `cborgen:"size" dagjsongen:"size"`
	Address *BlobAddress `cborgen:"address,omitempty" dagjsongen:"address,omitempty"`
}

type BlobAddress struct {
	URL     capabilities.CborURL `cborgen:"url" dagjsongen:"url"`
	Headers map[string]string    `cborgen:"headers" dagjsongen:"headers"`
	Expires int64                `cborgen:"expires" dagjsongen:"expires"`
}

type ListArguments struct {
	Cursor *string `cborgen:"cursor,omitempty" dagjsongen:"cursor,omitempty"`
	Size   *uint64 `cborgen:"size,omitempty" dagjsongen:"size,omitempty"`
}

type ListOK struct {
	Cursor  *string        `cborgen:"cursor,omitempty" dagjsongen:"cursor,omitempty"`
	Size    uint64         `cborgen:"size" dagjsongen:"size"`
	Results []ListBlobItem `cborgen:"results" dagjsongen:"results"`
}

type ListBlobItem struct {
	Blob       Blob  `cborgen:"blob" dagjsongen:"blob"`
	InsertedAt int64 `cborgen:"insertedAt" dagjsongen:"insertedAt"`
}

type RemoveArguments struct {
	Digest multihash.Multihash `cborgen:"digest" dagjsongen:"digest"`
}

type ReplicateArguments struct {
	// Blob is the blob that must be replicated.
	Blob Blob `cborgen:"blob" dagjsongen:"blob"`
	// Replicas is the number of replicas to ensure.
	// e.g. Replicas: 3 will ensure 3 copies of the data exist in a network in total.
	Replicas uint64 `cborgen:"replicas" dagjsongen:"replicas"`
	// Site is a link to a location commitment indicating where the Blob must be
	// fetched from.
	Site cid.Cid `cborgen:"site" dagjsongen:"site"`
}

type ReplicateOK struct {
	// Site resolves to additional locations for the blob. They are links to
	// `/blob/replica/transfer` tasks.
	Site []promise.AwaitOK `cborgen:"site" dagjsongen:"site"`
}

// RetrieveBlob identifies a blob solely by its content multihash. Used by the
// service-level `/blob/retrieve` capability where the caller is fetching data
// by hash without prior knowledge of the byte size.
type RetrieveBlob struct {
	Digest multihash.Multihash `cborgen:"digest" dagjsongen:"digest"`
}

// RetrieveArguments is the argument shape of the `/blob/retrieve` capability —
// a service-level (not space-scoped) retrieval handle. Compare to
// `content.RetrieveArguments` which is space-scoped and carries a byte Range;
// the `/blob/retrieve` flow is consumed by service principals (e.g. the
// indexer) fetching content claims that aren't bound to any space.
type RetrieveArguments struct {
	Blob RetrieveBlob `cborgen:"blob" dagjsongen:"blob"`
}

// RetrieveOK is the success return for `/blob/retrieve`. The blob bytes
// themselves are streamed back through the response container's body (the
// libforge HTTPHeader retrieval transport); the typed OK record is empty.
type RetrieveOK struct{}
