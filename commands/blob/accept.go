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

// AllocationExpiredErrorName is the stable receipt-failure name when the
// allocation passed its expiry before the content was delivered. The blob is
// allocated again to upload it.
const AllocationExpiredErrorName = "AllocationExpired"

// ErrAllocationExpired is the failure for an allocation that expired before
// the content was delivered.
var ErrAllocationExpired = errors.New(AllocationExpiredErrorName, "allocation expired before the content was delivered")

// AllocationNotFoundErrorName is the stable receipt-failure name when the
// storage node holds no allocation for the accepted blob: it was never made,
// was rejected, or was removed after it expired.
const AllocationNotFoundErrorName = "AllocationNotFound"

// ErrAllocationNotFound is the failure for an accept of a blob the storage
// node holds no allocation for.
var ErrAllocationNotFound = errors.New(AllocationNotFoundErrorName, "allocation not found")
