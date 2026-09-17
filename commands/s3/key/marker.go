//go:build !codegen

// Package key defines the `/s3/key/...` commands for tenant access keys.
//
// A principal-bound access key holds exactly one delegation: the marker, issued
// by the tenant to the key with the tenant as subject. It grants nothing,
// appears in no proof chain, and is never invoked. It exists so the key has a
// delegation CID that the gateway holds and Hilt can revoke through
// `/ucan/revoke`.
package key

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

// Marker is the `/s3/key/marker` command. Its arguments and result are both
// empty, because nothing ever invokes it.
var Marker = binding.Bind[*commands.Unit, *commands.Unit](command.MustParse("/s3/key/marker"))
