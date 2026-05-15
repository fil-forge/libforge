//go:build !codegen

package content

import "github.com/fil-forge/libforge/capabilities"

const RetrieveCommand = "/content/retrieve"

type RetrieveOK = capabilities.Unit

var Retrieve = capabilities.MustNew[*RetrieveArguments](RetrieveCommand)
