package blob

import (
	sbdm "github.com/alanshaw/libracha/capabilities/space/blob/datamodel"
	"github.com/alanshaw/ucantone/validator/bindcap"
)

const ListCommand = "/blob/list"

type (
	ListArguments = sbdm.ListArgumentsModel
	ListOK        = sbdm.ListOKModel
)

var List, _ = bindcap.New[*ListArguments](
	ListCommand,
)
