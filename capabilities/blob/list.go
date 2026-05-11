package blob

import (
	bdm "github.com/fil-forge/libforge/capabilities/blob/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

const ListCommand = "/blob/list"

type (
	ListArguments = bdm.ListArgumentsModel
	ListOK        = bdm.ListOKModel
	ListBlobItem  = bdm.ListBlobItem
)

var List, _ = bindcap.New[*ListArguments](ListCommand)
