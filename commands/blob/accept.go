//go:build !codegen

package blob

import "github.com/fil-forge/libforge/commands"

var Accept = commands.MustParse[*AcceptArguments]("/blob/accept")
