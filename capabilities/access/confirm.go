package access

import (
	adm "github.com/fil-forge/libforge/capabilities/access/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

type (
	ConfirmArguments = adm.ConfirmArgumentsModel
	ConfirmOK        = adm.ConfirmOKModel
)

const ConfirmCommand = "/access/confirm"

// Confirm can be invoked by an agent to confirm an access request.
var Confirm, _ = bindcap.New[*ConfirmArguments](ConfirmCommand)
