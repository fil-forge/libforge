//go:build !codegen

package access

import "github.com/fil-forge/libforge/commands"

type ClaimArguments = commands.Unit

// Claim can be invoked by an agent to claim a set of delegations from the
// account.
var Claim = commands.MustParse[*ClaimArguments]("/access/claim")
