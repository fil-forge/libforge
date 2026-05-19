//go:build !codegen

package access

import "github.com/fil-forge/libforge/commands"

type DelegateOK = commands.Unit

// Delegate can be invoked by an agent to delegate a set of capabilities that
// may be subsequently claimed by another agent.
var Delegate = commands.MustParse[*DelegateArguments]("/access/delegate")

const (
	DelegationNotFoundErrorName  = "DelegationNotFound"
	InsufficientStorageErrorName = "InsufficientStorage"
)
