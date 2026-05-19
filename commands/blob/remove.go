//go:build !codegen

package blob

import "github.com/fil-forge/libforge/commands"

type RemoveOK = commands.Unit

var Remove = commands.MustParse[*RemoveArguments]("/blob/remove")
