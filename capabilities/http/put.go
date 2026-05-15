//go:build !codegen

package http

import "github.com/fil-forge/libforge/capabilities"

const PutCommand = "/http/put"

type PutOK = capabilities.Unit

var Put = capabilities.MustNew[*PutArguments](PutCommand)
