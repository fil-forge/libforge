//go:build !codegen

package blob

import (
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/errors"
	"github.com/fil-forge/ucantone/ucan/command"
)

var Accept = binding.Bind[*AcceptArguments, *AcceptOK](command.MustParse("/blob/accept"))

// BlobDigestMismatchErrorName is the stable receipt-failure name when the
// digest the storage node computed for the received data differs from the
// digest in the accept arguments or, when they name only a digest code, the
// digest in the `/http/put` receipt.
const BlobDigestMismatchErrorName = "BlobDigestMismatch"

// ErrBlobDigestMismatch is the failure for a digest mismatch.
var ErrBlobDigestMismatch = errors.New(BlobDigestMismatchErrorName, "received data does not hash to the reported digest")
