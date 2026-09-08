//go:build !codegen

package routing

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

type PutOK = commands.Unit

// Put (/routing/put) replaces the candidate set of a routing policy. The
// subject is the policy DID. A policy needs no registration step: it exists in
// the upload service once its first put succeeds.
//
// Every candidate MUST identify a storage node registered with the upload
// service and the set MUST NOT be empty; otherwise the invocation fails with
// InvalidCandidates. A change applies to writes routed after it takes effect;
// in-flight writes MAY be routed using the previous set.
//
// The receipt carries no payload (Unit).
var Put = binding.Bind[*PutArguments, *PutOK](command.MustParse("/routing/put"))

// InvalidCandidatesErrorName is the stable receipt-failure name when a put
// carries an empty candidate set or a candidate that is not a registered
// storage node.
const InvalidCandidatesErrorName = "InvalidCandidates"
