package shard

import (
	dm "github.com/fil-forge/libforge/capabilities/upload/shard/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

const ListCommand = "/upload/shard/list"

type (
	ListArguments = dm.ListArgumentsModel
	ListOK        = dm.ListOKModel
)

var List, _ = bindcap.New[*ListArguments](ListCommand)
