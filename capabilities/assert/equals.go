//go:build !codegen

package assert

import "github.com/fil-forge/libforge/capabilities"

const EqualsCommand = "/assert/equals"

type EqualsOK = capabilities.Unit

// Equals claims data is referred to by another CID e.g CAR CID & Piece CID
var Equals = capabilities.MustNew[*EqualsArguments](EqualsCommand)
