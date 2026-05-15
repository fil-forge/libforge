//go:build !codegen

package access

import "github.com/fil-forge/libforge/capabilities"

const DelegateCommand = "/access/delegate"

type DelegateOK = capabilities.Unit

// Delegate can be invoked by an agent to delegate a set of capabilities that
// may be subsequently claimed by another agent.
var Delegate = capabilities.MustNew[*DelegateArguments](DelegateCommand)

const (
	DelegationNotFoundErrorName  = "DelegationNotFound"
	InsufficientStorageErrorName = "InsufficientStorage"
)
