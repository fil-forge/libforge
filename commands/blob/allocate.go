//go:build !codegen

package blob

import "github.com/fil-forge/libforge/commands"

const MaxBlobSize = 268_435_456

var Allocate = commands.MustParse[*AllocateArguments]("/blob/allocate")
