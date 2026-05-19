//go:build !codegen

package replica

import "github.com/fil-forge/libforge/commands"

var Allocate = commands.MustParse[*AllocateArguments]("/blob/replica/allocate")
