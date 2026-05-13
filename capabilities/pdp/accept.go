package pdp

import (
	pdm "github.com/fil-forge/libforge/capabilities/pdp/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

const AcceptCommand = "/pdp/accept"

type (
	AcceptArguments = pdm.AcceptArgumentsModel
	AcceptOK        = pdm.AcceptOKModel
)

var Accept, _ = bindcap.New[*AcceptArguments](AcceptCommand)
