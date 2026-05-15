//go:build !codegen

package attest

import "github.com/fil-forge/libforge/capabilities"

const ProofCommand = "/ucan/attest/proof"

type ProofOK = capabilities.Unit

// Issued by a trusted authority (usually the one handling invocation) that
// attests a specific UCAN delegation has been considered authentic.
var Proof = capabilities.MustNew[*ProofArguments](ProofCommand)
