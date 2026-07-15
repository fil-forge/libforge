//go:build !codegen

package blob

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

type RemoveOK = commands.Unit

// Remove (/blob/remove) releases a space's claim on a blob.
//
// Served by BOTH the upload service and storage nodes, at different levels:
// the upload service (subject = the space) validates the caller's space
// authority, deregisters the blob, and forwards the removal to every storage
// node holding it; a storage node (subject = the provider DID, invoked by the
// upload service under its registration delegation) drops the space's
// allocation/acceptance/claim and physically deletes the bytes only when no
// space claims the digest anymore (an accepted blob's bytes may additionally
// be retained until its PDP aggregate root is fully retired on-chain).
//
// Idempotent: removing an unknown or already-removed blob succeeds. The
// receipt carries no payload (Unit).
var Remove = binding.Bind[*RemoveArguments, *RemoveOK](command.MustParse("/blob/remove"))
