//go:build !codegen

package assert

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

type EqualsOK = commands.Unit

// Equals claims data is referred to by another CID e.g CAR CID & Piece CID
var Equals = binding.Bind[*EqualsArguments, *EqualsOK](command.MustParse("/assert/equals"))
