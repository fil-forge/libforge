//go:build !codegen

package blob

import (
	"github.com/fil-forge/libforge/capabilities"
	"github.com/fil-forge/ucantone/ucan/delegation/policy"
	"github.com/fil-forge/ucantone/validator/capability"
)

const AddCommand = "/blob/add"

var Add = capabilities.MustNew[*AddArguments](
	AddCommand,
	capability.WithPolicyBuilder(
		policy.GreaterThan(".blob.size", 0),
		policy.LessThanOrEqual(".blob.size", MaxBlobSize),
	),
)
