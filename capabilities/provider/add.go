package provider

import (
	dm "github.com/fil-forge/libforge/capabilities/provider/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

const AddCommand = "/provider/add"

type (
	AddArguments = dm.AddArgumentsModel
	AddOK        = dm.AddOKModel
)

var Add, _ = bindcap.New[*AddArguments](AddCommand)
