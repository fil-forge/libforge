//go:build !codegen

package egress

import "github.com/fil-forge/libforge/commands"

// Track is the capability a storage node invokes to ask the egress
// tracking service to record egress accounted for in a batch of
// `/content/retrieve` receipts. The tracking service responds by forking
// a `/space/egress/consolidate` sub-invocation onto the receipt's
// effects; the typed OK return is empty.
var Track = commands.MustParse[*TrackArguments]("/space/egress/track")
