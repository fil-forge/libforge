//go:build !codegen

package upload

import "github.com/fil-forge/libforge/commands"

type RemoveOK = commands.Unit

var Remove = commands.MustParse[*RemoveArguments]("/upload/remove")
