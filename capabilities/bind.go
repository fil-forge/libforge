package capabilities

import (
	"github.com/fil-forge/ucantone/ucan"
	"github.com/fil-forge/ucantone/validator/bindcap"
	"github.com/fil-forge/ucantone/validator/capability"
)

// MustNew is like [bindcap.New] but panics if the capability cannot be
// constructed. It exists for package-level capability declarations where the
// command and options are static — any error indicates a programming bug
// (malformed command string, invalid option) and should fail loudly at init
// rather than be silently dropped with `, _`.
func MustNew[A bindcap.Arguments](cmd ucan.Command, opts ...capability.Option) *bindcap.Capability[A] {
	c, err := bindcap.New[A](cmd, opts...)
	if err != nil {
		panic(err)
	}
	return c
}
