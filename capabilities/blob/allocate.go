//go:build !codegen

package blob

import (
	"github.com/fil-forge/libforge/capabilities"
	"github.com/fil-forge/ucantone/ucan/delegation/policy"
	"github.com/fil-forge/ucantone/validator/capability"
)

const MaxBlobSize = 268_435_456

const AllocateCommand = "/blob/allocate"

var Allocate = capabilities.MustNew[*AllocateArguments](
	AllocateCommand,
	capability.WithPolicyBuilder(
		policy.GreaterThan(".blob.size", 0),
		policy.LessThanOrEqual(".blob.size", MaxBlobSize),
	),
)
