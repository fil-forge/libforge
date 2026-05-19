//go:build !codegen

package provider

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/errors"
)

var Add = commands.MustParse[*AddArguments]("/provider/add")

const (
	InvalidAccountErrorName     = "InvalidAccount"
	AccountPlanMissingErrorName = "AccountPlanMissing"
)

var ErrAccountPlanMissing = errors.New(AccountPlanMissingErrorName, "account does not have an active payment plan")
