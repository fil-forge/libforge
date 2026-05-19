//go:build !codegen

package assert

import "github.com/fil-forge/libforge/commands"

type EqualsOK = commands.Unit

// Equals claims data is referred to by another CID e.g CAR CID & Piece CID
var Equals = commands.MustParse[*EqualsArguments]("/assert/equals")
