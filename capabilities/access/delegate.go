package access

import (
	adm "github.com/fil-forge/libforge/capabilities/access/datamodel"
	cdm "github.com/fil-forge/libforge/capabilities/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

type (
	DelegateArguments = adm.DelegateArgumentsModel
	DelegateOK        = cdm.UnitModel
)

const DelegateCommand = "/access/delegate"

// Delegate can be invoked by an agent to delegate a set of capabilities that
// may be subsequently claimed by another agent.
var Delegate, _ = bindcap.New[*DelegateArguments](DelegateCommand)

const (
	DelegationNotFoundErrorName  = "DelegationNotFound"
	InsufficientStorageErrorName = "InsufficientStorage"
)
