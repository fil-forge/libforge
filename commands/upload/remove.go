//go:build !codegen

package upload

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

type RemoveOK = commands.Unit

var Remove = binding.Bind[*RemoveArguments, *RemoveOK](command.MustParse("/upload/remove"))
