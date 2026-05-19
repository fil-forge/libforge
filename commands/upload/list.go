//go:build !codegen

package upload

import "github.com/fil-forge/libforge/commands"

var List = commands.MustParse[*ListArguments]("/upload/list")
