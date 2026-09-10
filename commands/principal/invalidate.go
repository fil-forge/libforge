//go:build !codegen

package principal

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

type InvalidateOK = commands.Unit

// Invalidate is the `/principal/invalidate` command. Hilt invokes it on Swarf
// as its service identity before it commits a change to what a principal can
// reach, recording that every proof a gateway cached for that principal's
// access keys is void. No delegation names a principal, so there is no proof
// chain to carry the authority: Swarf accepts the command only from issuers in
// its publisher list.
var Invalidate = binding.Bind[*InvalidateArguments, *InvalidateOK](command.MustParse("/principal/invalidate"))
