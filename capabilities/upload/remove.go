//go:build !codegen

package upload

import "github.com/fil-forge/libforge/capabilities"

const RemoveCommand = "/upload/remove"

type RemoveOK = capabilities.Unit

var Remove = capabilities.MustNew[*RemoveArguments](RemoveCommand)
