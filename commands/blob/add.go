//go:build !codegen

package blob

import "github.com/fil-forge/libforge/commands"

var Add = commands.MustParse[*AddArguments]("/blob/add")
