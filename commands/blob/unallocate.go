//go:build !codegen

package blob

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

type UnallocateOK = commands.Unit

// Unallocate (/blob/unallocate) retires a PARKED blob — one that was
// allocated and uploaded (HTTP PUT) but never accepted. It is the abandon
// half of the blob lifecycle: an upload ends in exactly one of
// `/blob/accept` (commit: aggregation, location claim, registration) or
// `/blob/unallocate` (drop: allocation released, bytes deleted).
//
// Served by BOTH the upload service and storage nodes, at different levels:
// the upload service (subject = the space) recovers the storage node holding
// the parked blob from the Cause receipt chain and forwards the unallocate;
// a storage node (subject = the provider DID, invoked by the upload service
// under its registration delegation) drops the space's allocation and
// deletes the bytes once no space holds an allocation. A blob with any
// acceptance is refused — accepted blobs are released via `/blob/remove`.
//
// Idempotent: unallocating an unknown or already-unallocated blob succeeds.
// The receipt carries no payload (Unit).
var Unallocate = binding.Bind[*UnallocateArguments, *UnallocateOK](command.MustParse("/blob/unallocate"))
