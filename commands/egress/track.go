//go:build !codegen

package egress

import (
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

// Track is the command a reporter invokes on the egress tracking service to
// record the egress accounted for in a batch of receipts.
//
// A reporter may submit one batch under more than one track task: it issues a
// new invocation when it cannot tell whether the last one was accepted, and
// when a consolidation fails. The tracker must count a batch CID at most once
// for a reporter, however many tasks name it.
var Track = binding.Bind[*TrackArguments, *TrackOK](command.MustParse("/egress/track"))
