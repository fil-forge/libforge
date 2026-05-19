//go:build !codegen

package assert

import "github.com/fil-forge/libforge/commands"

type IndexOK = commands.Unit

// Index claims that a content graph can be found in blob(s) that are identified
// and indexed in the given index CID.
var Index = commands.MustParse[*IndexArguments]("/assert/index")
