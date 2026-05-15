//go:build !codegen

package upload

import "github.com/fil-forge/libforge/capabilities"

const AddCommand = "/upload/add"

type AddOK = capabilities.Unit

var Add = capabilities.MustNew[*AddArguments](AddCommand)
