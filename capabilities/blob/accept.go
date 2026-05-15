//go:build !codegen

package blob

import "github.com/fil-forge/libforge/capabilities"

const AcceptCommand = "/blob/accept"

var Accept = capabilities.MustNew[*AcceptArguments](AcceptCommand)
