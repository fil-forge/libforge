//go:build !codegen

package debug

import (
	"github.com/fil-forge/libforge/capabilities"
	"github.com/fil-forge/ucantone/ucan/delegation/policy"
	"github.com/fil-forge/ucantone/validator/capability"
)

const EchoCommand = "/debug/echo"

type EchoOK = EchoArguments

var Echo = capabilities.MustNew[*EchoArguments](
	EchoCommand,
	capability.WithPolicyBuilder(policy.NotEqual(".message", "")),
)
