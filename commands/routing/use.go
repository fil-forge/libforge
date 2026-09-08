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
// The space MUST be provisioned with a provider (SpaceNotProvisioned) and the
// policy MUST be known to the upload service, i.e. have a stored candidate set
// (UnknownPolicy).
//
// The receipt carries no payload (Unit).
var Use = binding.Bind[*UseArguments, *UseOK](command.MustParse("/routing/use"))

const (
	// UnknownPolicyErrorName is the stable receipt-failure name when the
	// referenced policy has no stored candidate set.
	UnknownPolicyErrorName = "UnknownPolicy"
	// SpaceNotProvisionedErrorName is the stable receipt-failure name when the
	// subject space has no service provider.
	SpaceNotProvisionedErrorName = "SpaceNotProvisioned"
)
