//go:build !codegen

package attest

import "github.com/fil-forge/libforge/commands"

type ProofOK = commands.Unit

// Issued by a trusted authority (usually the one handling invocation) that
// attests a specific UCAN delegation has been considered authentic.
var Proof = commands.MustParse[*ProofArguments]("/ucan/attest/proof")
