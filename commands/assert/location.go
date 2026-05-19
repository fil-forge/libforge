//go:build !codegen

package assert

import "github.com/fil-forge/libforge/commands"

type LocationOK = commands.Unit

var Location = commands.MustParse[*LocationArguments]("/assert/location")
