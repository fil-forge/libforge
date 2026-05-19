//go:build !codegen

package claim

import "github.com/fil-forge/libforge/commands"

type CacheOK = commands.Unit

var Cache = commands.MustParse[*CacheArguments]("/claim/cache")
