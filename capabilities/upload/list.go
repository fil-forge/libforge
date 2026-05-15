//go:build !codegen

package upload

import "github.com/fil-forge/libforge/capabilities"

const ListCommand = "/upload/list"

var List = capabilities.MustNew[*ListArguments](ListCommand)
