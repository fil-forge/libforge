//go:build !codegen

package egress

import (
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

// Track is the command a reporter invokes on the egress tracking service to
// record the egress accounted for in a batch of receipts.
var Track = binding.Bind[*TrackArguments, *TrackOK](command.MustParse("/egress/track"))
