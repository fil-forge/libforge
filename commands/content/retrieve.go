//go:build !codegen

package content

import "github.com/fil-forge/libforge/commands"

type RetrieveOK = commands.Unit

var Retrieve = commands.MustParse[*RetrieveArguments]("/content/retrieve")
