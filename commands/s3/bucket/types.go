package bucket

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/libforge/commands/s3"
	"github.com/fil-forge/ucantone/did"
)

// CreateArguments are the arguments to `/s3/bucket/create`.
type CreateArguments struct {
	// Request is the AWS S3 CreateBucket request.
	Request s3.Request `cborgen:"request" dagjsongen:"request"`
}

// DeleteArguments are the arguments to `/s3/bucket/delete`.
type DeleteArguments struct {
	// Request is the AWS S3 DeleteBucket request.
	Request s3.Request `cborgen:"request" dagjsongen:"request"`
}

// ListArguments are the arguments to `/s3/bucket/list`.
type ListArguments struct {
	// Request is the AWS S3 ListBuckets request.
	Request s3.Request `cborgen:"request" dagjsongen:"request"`
}

// PolicyArguments are the arguments to `/s3/bucket/policy`.
type PolicyArguments struct {
	// Request is the AWS S3 GetBucketPolicy, PutBucketPolicy or
	// DeleteBucketPolicy request.
	Request s3.Request `cborgen:"request" dagjsongen:"request"`
	// Body is the policy document a PutBucketPolicy request carries. It is
	// empty for the other two operations.
	Body []byte `cborgen:"body" dagjsongen:"body"`
}

// PolicyOK is the successful result of `/s3/bucket/policy`. The policy's
// entity tag is not carried: both sides compute it from the document with
// [ETag].
type PolicyOK struct {
	// Policy is the stored policy document. It is set only for a
	// GetBucketPolicy.
	Policy *PolicyDocument `cborgen:"policy,omitempty" dagjsongen:"policy,omitempty"`
}

// PolicyDocument is a bucket policy document. Its JSON encoding is the AWS
// bucket policy document; its CBOR and DAG-JSON encodings are hand-written
// (policy_codec.go) so that the statements go out in canonical order.
type PolicyDocument struct {
	Version    string       `json:"Version"`
	Statements StatementSet `json:"Statement"`
}

// StatementSet is a policy's statements: any order in memory, canonical on
// the wire. Every encoding (JSON, DAG-JSON, CBOR) writes them deduplicated
// and ordered by [StatementCID], so two documents that differ only in
// statement order or duplicates encode the same.
type StatementSet []Statement

// Effect is whether a [Statement] grants or withholds its actions.
type Effect string

const (
	Allow Effect = "Allow"
	Deny  Effect = "Deny"
)

// Statement grants (Allow) or withholds (Deny) actions on the bucket's
// resources to principals. Sid is an optional label the caller chooses. On
// the wire (CBOR and DAG-JSON) the three sets are maps from member to
// [commands.Unit]; in the AWS JSON document they are lists (see
// policy_json.go).
type Statement struct {
	Sid       string    `cborgen:"sid,omitempty" dagjsongen:"sid,omitempty" json:"Sid,omitempty"`
	Effect    Effect    `cborgen:"effect" dagjsongen:"effect" json:"Effect"`
	Principal Principal `cborgen:"principal" dagjsongen:"principal" json:"Principal"`
	Actions   StringSet `cborgen:"action" dagjsongen:"action" json:"Action"`
	Resources StringSet `cborgen:"resource" dagjsongen:"resource" json:"Resource"`
}

// StringSet is a set of strings: a map from each member to [commands.Unit],
// which the IPLD codecs write with sorted keys. Its JSON form is the sorted
// list of its members.
type StringSet map[string]commands.Unit

// Principal is the set of principal ids a statement applies to, or the
// single member [Wildcard] for every principal of the tenant. Its JSON
// encoding is the string "*" or {"AWS": [user ARNs]} (see policy_json.go).
type Principal StringSet

// InfoArguments are the arguments to `/s3/bucket/info`.
type InfoArguments struct {
	// Name is the global bucket name to look up.
	Name string `cborgen:"name" dagjsongen:"name"`
	// AccessKey is the access key DID the returned delegation chain should
	// terminate at.
	AccessKey did.DID `cborgen:"accessKey" dagjsongen:"accessKey"`
}

// InfoOK is the successful result of `/s3/bucket/info`. Its Permissions and
// Delegations fields are slice-valued maps wrapped in struct types
// ([s3.PermissionSet] etc.) with their own codecs, so this struct is generated
// normally (the generators delegate to a struct field's codec).
type InfoOK struct {
	// ID is the DID of the bucket.
	ID did.DID `cborgen:"id" dagjsongen:"id"`
	// Permissions maps the access key DID to its assigned S3 permissions.
	Permissions s3.PermissionSet `cborgen:"permissions" dagjsongen:"permissions"`
	// Delegations maps the CID of a delegation whose audience is the access key
	// to its proof chain from the bucket.
	Delegations s3.ProofSet `cborgen:"delegations" dagjsongen:"delegations"`
}

// ListOK is the successful result of `/s3/bucket/list`.
type ListOK struct {
	Buckets           []Bucket `cborgen:"buckets" dagjsongen:"buckets"`
	ContinuationToken string   `cborgen:"continuationToken" dagjsongen:"continuationToken"`
	Owner             Owner    `cborgen:"owner" dagjsongen:"owner"`
	Prefix            string   `cborgen:"prefix" dagjsongen:"prefix"`
}

// Bucket describes a single bucket in a `/s3/bucket/list` result.
type Bucket struct {
	ARN          string `cborgen:"arn" dagjsongen:"arn"`
	Region       string `cborgen:"region" dagjsongen:"region"`
	CreationDate string `cborgen:"creationDate" dagjsongen:"creationDate"`
	Name         string `cborgen:"name" dagjsongen:"name"`
}

// Owner is the owner of a set of buckets.
type Owner struct {
	DisplayName string `cborgen:"displayName" dagjsongen:"displayName"`
	ID          string `cborgen:"id" dagjsongen:"id"`
}
