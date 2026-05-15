//go:build !codegen

package access

import "github.com/fil-forge/libforge/capabilities"

const ClaimCommand = "/access/claim"

type ClaimArguments = capabilities.Unit

// Claim can be invoked by an agent to claim a set of delegations from the
// account.
var Claim = capabilities.MustNew[*ClaimArguments](ClaimCommand)
