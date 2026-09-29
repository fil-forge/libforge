package http

import (
	"github.com/fil-forge/libforge/commands/blob"
	"github.com/fil-forge/ucantone/ucan/promise"
	"github.com/multiformats/go-multihash"
)

type PutArguments struct {
	Body blob.BlobSpec `cborgen:"body" dagjsongen:"body"`
	// Destination is the promise that resolves to the upload destination
	// where the blob should be PUT to. It is the result of a /blob/allocate task.
	Destination promise.AwaitOK `cborgen:"destination" dagjsongen:"destination"`
}

// PutOK is the result of a successful `/http/put`. Blob is set when the
// `/blob/allocate` task named only a digest code: it carries the digest the
// sender computed as it sent the data, which the storage node checks against
// its own at `/blob/accept`.
type PutOK struct {
	Blob *PutBlob `cborgen:"blob,omitempty" dagjsongen:"blob,omitempty"`
}

// PutBlob is the blob a `/http/put` delivered.
type PutBlob struct {
	Digest multihash.Multihash `cborgen:"digest" dagjsongen:"digest"`
}
