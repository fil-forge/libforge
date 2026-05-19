package commands

import "github.com/fil-forge/ucantone/validator/bindcom"

// MustParse is like [bindcom.Parse] but panics if the command cannot be
// constructed. It exists for package-level command declarations where the
// command and options are static — any error indicates a programming bug
// (malformed command string, invalid option) and should fail loudly at init
// rather than be silently dropped with `, _`.
func MustParse[A bindcom.Arguments](cmd string) bindcom.Command[A] {
	c, err := bindcom.Parse[A](cmd)
	if err != nil {
		panic(err)
	}
	return c
}
