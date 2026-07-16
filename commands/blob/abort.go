//go:build !codegen

package blob

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

type AbortOK = commands.Unit

// Abort (/blob/abort) abandons an in-flight upload of a PARKED blob — one
// that was allocated and uploaded (HTTP PUT) but never accepted. It is the
// client-facing abandon verb: an upload ends in exactly one of
// `/blob/accept` (commit) or an abort that the upload service translates
// into `/blob/reject` on the storage node holding the blob.
//
// Served by the upload service (subject = the space). A parked blob has no
// registration or acceptance to look the storage node up by, so the service
// recovers it from the Cause receipt chain and forwards a `/blob/reject`.
// Blobs with an acceptance are released via `/blob/remove` instead.
//
// Idempotent: aborting an unknown or already-rejected blob succeeds.
// The receipt carries no payload (Unit).
var Abort = binding.Bind[*AbortArguments, *AbortOK](command.MustParse("/blob/abort"))
