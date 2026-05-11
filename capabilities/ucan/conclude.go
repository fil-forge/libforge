package ucan

import (
	cdm "github.com/fil-forge/libforge/capabilities/datamodel"
	udm "github.com/fil-forge/libforge/capabilities/ucan/datamodel"
	"github.com/fil-forge/ucantone/errors"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

type (
	ConcludeArguments = udm.ConcludeArgumentsModel
	ConcludeOK        = cdm.UnitModel
)

const ConcludeCommand = "/ucan/conclude"

var Conclude, _ = bindcap.New[*ConcludeArguments](ConcludeCommand)

const ConclusionReceiptNotFoundErrorName = "ConclusionReceiptNotFound"

var ErrConclusionReceiptNotFound = errors.New(ConclusionReceiptNotFoundErrorName, "conclusion receipt not found")
