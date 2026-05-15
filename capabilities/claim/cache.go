//go:build !codegen

package claim

import "github.com/fil-forge/libforge/capabilities"

const CacheCommand = "/claim/cache"

type CacheOK = capabilities.Unit

var Cache = capabilities.MustNew[*CacheArguments](CacheCommand)
