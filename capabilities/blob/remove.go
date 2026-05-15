//go:build !codegen

package blob

import "github.com/fil-forge/libforge/capabilities"

const RemoveCommand = "/blob/remove"

type RemoveOK = capabilities.Unit

var Remove = capabilities.MustNew[*RemoveArguments](RemoveCommand)
