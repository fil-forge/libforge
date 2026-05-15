//go:build !codegen

package shard

import "github.com/fil-forge/libforge/capabilities"

const ListCommand = "/upload/shard/list"

var List = capabilities.MustNew[*ListArguments](ListCommand)
