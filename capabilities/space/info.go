package space

import (
	cdm "github.com/fil-forge/libforge/capabilities/datamodel"
	dm "github.com/fil-forge/libforge/capabilities/space/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

const InfoCommand = "/space/info"

type (
	InfoArguments = cdm.UnitModel
	InfoOK        = dm.InfoOKModel
)

var Info, _ = bindcap.New[*InfoArguments](InfoCommand)

const UnknownSpaceErrorName = "UnknownSpace"
