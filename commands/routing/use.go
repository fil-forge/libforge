//go:build !codegen

package routing

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

type UseOK = commands.Unit

// Use (/routing/use) sets or clears the routing policy a space references. The
// subject is the space. With a policy, every subsequent `/blob/add` on the
// space is routed to one of the policy's candidates and fails with
// CandidateUnavailable when none can serve it. Without a policy the reference
// is cleared and the space returns to default routing.
//
// The space MUST be provisioned with a provider (InsufficientStorage, see the
// access package) and the policy MUST be known to the upload service, i.e. have
// a stored candidate set (UnknownPolicy).
//
// The receipt carries no payload (Unit).
var Use = binding.Bind[*UseArguments, *UseOK](command.MustParse("/routing/use"))

// UnknownPolicyErrorName is the stable receipt-failure name when the
// referenced policy has no stored candidate set. A space with no service
// provider fails with the shared InsufficientStorage name from the access
// package, as every other capability on an unprovisioned space does.
const UnknownPolicyErrorName = "UnknownPolicy"
