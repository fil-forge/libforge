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

// ServeOK is the success return for `/content/serve`.
type ServeOK struct {
	// Bytes is how many bytes of the range were sent to the client.
	Bytes uint64 `cborgen:"bytes" dagjsongen:"bytes"`
}
