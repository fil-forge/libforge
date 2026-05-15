//go:build !codegen

package blob

import "github.com/fil-forge/libforge/capabilities"

const ListCommand = "/blob/list"

var List = capabilities.MustNew[*ListArguments](ListCommand)
