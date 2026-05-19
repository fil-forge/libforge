//go:build !codegen

package http

import "github.com/fil-forge/libforge/commands"

type PutOK = commands.Unit

var Put = commands.MustParse[*PutArguments]("/http/put")
