//go:build !codegen

package blob

import (
	"github.com/fil-forge/libforge/capabilities"
	"github.com/fil-forge/ucantone/ucan/delegation/policy"
	"github.com/fil-forge/ucantone/validator/capability"
)

const ReplicateCommand = "/blob/replicate"

// Replicate is a capability that allows an agent to replicate a Blob into a
// space identified by did:key in the `with` field.
//
// A Replicate capability may only be invoked after a `/blob/accept` receipt has
// been receieved, indicating the source node has successfully received the blob.
// Each Replicate task MUST target a different node, and they MUST NOT target
// the original upload target.
//
// The Replicate task receipt includes async tasks for `/blob/replica/allocate`
// and `/blob/replica/transfer`. Successful completion of the
// `/blob/replica/transfer` task indicates the replication target has
// transferred and stored the blob. The number of `/blob/replica/allocate` and
// `/blob/replica/transfer` tasks corresponds directly to number of replicas
// requested.
var Replicate = capabilities.MustNew[*ReplicateArguments](
	ReplicateCommand,
	capability.WithPolicyBuilder(
		policy.GreaterThan(".blob.size", 0),
		policy.LessThanOrEqual(".blob.size", MaxBlobSize),
		policy.GreaterThanOrEqual(".replicas", 1),
	),
)
