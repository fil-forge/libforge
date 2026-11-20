//go:build !codegen

package egress

import (
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

// Consolidate is the command the egress tracking service invokes to process a
// tracked batch of receipts into egress totals.
var Consolidate = binding.Bind[*ConsolidateArguments, *ConsolidateOK](command.MustParse("/egress/consolidate"))
