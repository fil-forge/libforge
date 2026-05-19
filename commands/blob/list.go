//go:build !codegen

package blob

import "github.com/fil-forge/libforge/commands"

var List = commands.MustParse[*ListArguments]("/blob/list")
