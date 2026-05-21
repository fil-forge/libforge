//go:build !codegen

package access

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/binding"
	command "github.com/fil-forge/ucantone/ucan/command"
)

type ClaimArguments = commands.Unit

// Claim can be invoked by an agent to claim a set of delegations from the
// account.
var Claim = binding.Bind[*ClaimArguments, *ClaimOK](command.MustParse("/access/claim"))
