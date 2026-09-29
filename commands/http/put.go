//go:build !codegen

package http

import (
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

var Put = binding.Bind[*PutArguments, *PutOK](command.MustParse("/http/put"))
