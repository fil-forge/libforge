//go:build !codegen

package content

import (
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

// Serve is the command a gateway invokes on itself when it sends blob bytes to
// its client, from its own copy or from a storage node. The invocation says
// what was asked for and the receipt says how much was sent.
var Serve = binding.Bind[*ServeArguments, *ServeOK](command.MustParse("/content/serve"))
