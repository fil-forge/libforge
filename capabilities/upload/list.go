package upload

import (
	dm "github.com/fil-forge/libforge/capabilities/upload/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

const ListCommand = "/upload/list"

type (
	ListArguments  = dm.ListArgumentsModel
	ListOK         = dm.ListOKModel
	ListUploadItem = dm.ListUploadItem
)

var List, _ = bindcap.New[*ListArguments](ListCommand)
