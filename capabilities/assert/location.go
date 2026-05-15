//go:build !codegen

package assert

import "github.com/fil-forge/libforge/capabilities"

const LocationCommand = "/assert/location"

type LocationOK = capabilities.Unit

var Location = capabilities.MustNew[*LocationArguments](LocationCommand)
