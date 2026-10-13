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

// PolicyOK is the successful result of `/s3/bucket/policy`.
type PolicyOK struct {
	// ETag is the tag of the bucket's policy after the operation. It is empty
	// after a DeleteBucketPolicy.
	ETag string `cborgen:"etag" dagjsongen:"etag"`
	// Policy is the stored policy document. It is set only for a
	// GetBucketPolicy.
	Policy *BucketPolicy `cborgen:"policy,omitempty" dagjsongen:"policy,omitempty"`
}

// BucketPolicy is a bucket policy document. Its JSON encoding is the AWS
// bucket policy document.
type BucketPolicy struct {
	Version    string      `cborgen:"version" dagjsongen:"version" json:"Version"`
	Statements []Statement `cborgen:"statement" dagjsongen:"statement" json:"Statement"`
}

// Effect is whether a [Statement] grants or withholds its actions.
type Effect string

const (
	Allow Effect = "Allow"
	Deny  Effect = "Deny"
)

// Statement grants (Allow) or withholds (Deny) actions on the bucket's
// resources to principals. Sid is an optional label the caller chooses.
type Statement struct {
	Sid       string    `json:"Sid,omitempty"`
	Effect    Effect    `json:"Effect"`
	Principal Principal `json:"Principal"`
	Actions   []string  `json:"Action"`
	Resources []string  `json:"Resource"`
}

// StatementModel is the IPLD form of a [Statement]: its principal, action
// and resource lists as sets, a set being a map from its members to
// [commands.Unit], and a wildcard principal the single member [Wildcard].
// [Statement] encodes as it in CBOR and DAG-JSON (see policy_model.go).
type StatementModel struct {
	Sid       string                   `cborgen:"sid,omitempty" dagjsongen:"sid,omitempty"`
	Effect    Effect                   `cborgen:"effect" dagjsongen:"effect"`
	Principal map[string]commands.Unit `cborgen:"principal" dagjsongen:"principal"`
	Action    map[string]commands.Unit `cborgen:"action" dagjsongen:"action"`
	Resource  map[string]commands.Unit `cborgen:"resource" dagjsongen:"resource"`
}

// Principal is the set of principals a statement applies to: every principal
// of the tenant (All), or the listed ids. Its JSON encoding is the string "*"
// or {"AWS": [user ARNs]} (see policy_json.go).
type Principal struct {
	All bool
	IDs []string
}

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
