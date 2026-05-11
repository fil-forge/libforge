package provider

import (
	dm "github.com/fil-forge/libforge/capabilities/provider/datamodel"
	"github.com/fil-forge/ucantone/errors"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

const AddCommand = "/provider/add"

type (
	AddArguments = dm.AddArgumentsModel
	AddOK        = dm.AddOKModel
)

var Add, _ = bindcap.New[*AddArguments](AddCommand)

const (
	InvalidAccountErrorName     = "InvalidAccount"
	AccountPlanMissingErrorName = "AccountPlanMissing"
)

var ErrAccountPlanMissing = errors.New(AccountPlanMissingErrorName, "account does not have an active payment plan")
