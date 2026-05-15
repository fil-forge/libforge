//go:build !codegen

package provider

import (
	"github.com/fil-forge/libforge/capabilities"
	"github.com/fil-forge/ucantone/errors"
)

const AddCommand = "/provider/add"

var Add = capabilities.MustNew[*AddArguments](AddCommand)

const (
	InvalidAccountErrorName     = "InvalidAccount"
	AccountPlanMissingErrorName = "AccountPlanMissing"
)

var ErrAccountPlanMissing = errors.New(AccountPlanMissingErrorName, "account does not have an active payment plan")
