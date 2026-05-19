//go:build !codegen

package debug

import "github.com/fil-forge/libforge/commands"

type EchoOK = EchoArguments

var Echo = commands.MustParse[*EchoArguments]("/debug/echo")
