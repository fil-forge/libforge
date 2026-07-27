//go:build !codegen

package blob

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

type ReleaseOK = commands.Unit

// Release (/blob/release) is the claim-release inverse of `/blob/allocate`:
// it drops one space's reference to a blob on a storage node. It is the
// upload service's translation of a client `/blob/remove`.
//
// Served by storage nodes (subject = the provider DID, invoked by the upload
// service under its registration delegation; the space travels in the
// arguments, matching Allocate/Accept). The node drops the space's
// allocation, acceptance and location claim. Bytes are physically deleted
// only when no space claims the digest anymore — and an accepted blob's
// bytes may additionally be retained until its PDP aggregate root is fully
// retired on-chain. Physical deletion is always asynchronous: the removal
// machinery re-verifies zero claims before every destructive step.
//
// Idempotent: releasing an unknown or already-released blob succeeds. The
// receipt carries no payload (Unit).
var Release = binding.Bind[*ReleaseArguments, *ReleaseOK](command.MustParse("/blob/release"))
