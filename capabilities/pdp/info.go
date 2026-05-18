//go:build !codegen

package pdp

import "github.com/fil-forge/libforge/capabilities"

const InfoCommand = "/pdp/info"

var Info = capabilities.MustNew[*InfoArguments](InfoCommand)
