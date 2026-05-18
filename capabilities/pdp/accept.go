//go:build !codegen

package pdp

import "github.com/fil-forge/libforge/capabilities"

const AcceptCommand = "/pdp/accept"

var Accept = capabilities.MustNew[*AcceptArguments](AcceptCommand)
