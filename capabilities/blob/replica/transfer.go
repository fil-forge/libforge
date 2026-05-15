//go:build !codegen

package replica

import (
	"github.com/fil-forge/libforge/capabilities"
	"github.com/fil-forge/libforge/capabilities/blob"
	"github.com/fil-forge/ucantone/ucan/delegation/policy"
	"github.com/fil-forge/ucantone/validator/capability"
)

const TransferCommand = "/blob/replica/transfer"

var Transfer = capabilities.MustNew[*TransferArguments](
	TransferCommand,
	capability.WithPolicyBuilder(
		policy.GreaterThan(".blob.size", 0),
		policy.LessThanOrEqual(".blob.size", blob.MaxBlobSize),
	),
)
