//go:build !codegen

package space

import "github.com/fil-forge/libforge/capabilities"

const InfoCommand = "/space/info"

type InfoArguments = capabilities.Unit

var Info = capabilities.MustNew[*InfoArguments](InfoCommand)

const UnknownSpaceErrorName = "UnknownSpace"
