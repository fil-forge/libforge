package content

import "github.com/multiformats/go-multihash"

type Blob struct {
	Digest multihash.Multihash `cborgen:"digest" dagjsongen:"digest"`
}

type Range struct {
	Start uint64 `cborgen:"start" dagjsongen:"start"`
	End   uint64 `cborgen:"end" dagjsongen:"end"`
}

type RetrieveArguments struct {
	Blob  Blob  `cborgen:"blob" dagjsongen:"blob"`
	Range Range `cborgen:"range" dagjsongen:"range"`
}

// ServeArguments is the argument shape of `/content/serve`.
type ServeArguments struct {
	// Blob is the blob the bytes are served from.
	Blob Blob `cborgen:"blob" dagjsongen:"blob"`
	// Range is the first and last byte of the blob requested, both inclusive.
	Range Range `cborgen:"range" dagjsongen:"range"`
}

// ServeOK is the success return for `/content/serve`. Its two counts differ
// when the blob's stored bytes are not the bytes the client receives, as for
// an encrypted blob.
type ServeOK struct {
	// Read is how many stored bytes of the range were read from the blob. It
	// is in the unit of the invocation's range, and is at most its size.
	Read uint64 `cborgen:"read" dagjsongen:"read"`
	// Sent is how many bytes were sent to the client from the bytes read. It
	// is at most Read.
	Sent uint64 `cborgen:"sent" dagjsongen:"sent"`
}
