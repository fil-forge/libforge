package pdp

import (
	pdm "github.com/fil-forge/libforge/capabilities/pdp/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

const InfoCommand = "/pdp/info"

type (
	InfoArguments          = pdm.InfoArgumentsModel
	InfoOK                 = pdm.InfoOKModel
	InfoAcceptedAggregate = pdm.InfoAcceptedAggregateModel
)

var Info, _ = bindcap.New[*InfoArguments](InfoCommand)
