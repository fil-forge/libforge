//go:build !codegen

package shard

import "github.com/fil-forge/libforge/commands"

var List = commands.MustParse[*ListArguments]("/upload/shard/list")
