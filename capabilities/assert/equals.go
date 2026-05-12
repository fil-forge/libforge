package assert

import (
	adm "github.com/fil-forge/libforge/capabilities/assert/datamodel"
	cdm "github.com/fil-forge/libforge/capabilities/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

type (
	EqualsArguments = adm.EqualsArgumentsModel
	EqualsOK        = cdm.UnitModel
)

const EqualsCommand = "/assert/equals"

// Equals claims data is referred to by another CID e.g CAR CID & Piece CID
var Equals, _ = bindcap.New[*EqualsArguments](EqualsCommand)
