//go:build !codegen

package blob

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

type RejectOK = commands.Unit

// Reject (/blob/reject) is the "don't accept" inverse of `/blob/accept`: it
// retires a PARKED blob — allocated and uploaded (HTTP PUT), never accepted.
// A blob's lifecycle on a storage node ends in exactly one of
// `/blob/accept` (commit: aggregation, location claim, registration) or
// `/blob/reject` (drop: allocation released, bytes deleted).
//
// Served by storage nodes (subject = the provider DID, invoked by the
// upload service under its registration delegation, typically translating a
// client `/blob/abort`). The node drops the space's allocation and deletes
// the bytes once no space holds an allocation. A blob with any acceptance
// is refused — accepted blobs are released via `/blob/remove`.
//
// Idempotent: rejecting an unknown or already-rejected blob succeeds.
// The receipt carries no payload (Unit).
var Reject = binding.Bind[*RejectArguments, *RejectOK](command.MustParse("/blob/reject"))
