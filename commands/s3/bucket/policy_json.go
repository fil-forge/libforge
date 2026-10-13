package bucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/fil-forge/ucantone/errors"
)

// Version2012 and Version2008 are the policy language versions a document
// may declare. Forge evaluates both the same way.
const (
	Version2012 = "2012-10-17"
	Version2008 = "2008-10-17"
)

// Wildcard is the principal "*", naming every principal of the tenant. It is
// carried as the bare JSON string "*" or as {"AWS": "*"}; inside a list it is
// rejected.
const Wildcard = "*"

// PolicyWildcard is the action "s3:*", standing for every policy action.
const PolicyWildcard = "s3:*"

// UserARNPrefix is the prefix of a principal's user ARN; the principal id
// follows it.
const UserARNPrefix = "arn:aws:iam:::user/"

// s3ARNPrefix and objectsSuffix make up the policy bucket's ARNs, the only
// resources a document names.
const (
	s3ARNPrefix   = "arn:aws:s3:::"
	objectsSuffix = "/*"
)

// MaxSetSize bounds the statements of a policy and the principals of a
// statement. A real policy is far smaller.
const MaxSetSize = 4096

// policyActions is the allowlist of permissions a bucket policy statement may
// carry, sorted for [PolicyAction]'s binary search. The rest of the S3
// permission vocabulary stays out on purpose: a key holding s3:CreateBucket
// or s3:DeleteBucket acts outside the policy that granted it, a key holding a
// policy operation could rewrite the policy that granted it, and every
// principal holds s3:ListAllMyBuckets.
var policyActions = sorted([]string{
	"s3:AbortMultipartUpload",
	"s3:DeleteObject",
	"s3:DeleteObjectVersion",
	"s3:GetObject",
	"s3:GetObjectLegalHold",
	"s3:GetObjectRetention",
	"s3:GetObjectVersion",
	"s3:ListBucket",
	"s3:ListBucketMultipartUploads",
	"s3:ListBucketVersions",
	"s3:ListMultipartUploadParts",
	"s3:PutObject",
	"s3:PutObjectLegalHold",
	"s3:PutObjectRetention",
})

func sorted(s []string) []string {
	slices.Sort(s)
	return s
}

// bucketActions are the policy actions that act on the bucket rather than on
// its objects; every other policy action is an object action. The split is
// versitygw's, which evaluates the policy against these ARNs.
var bucketActions = []string{
	"s3:ListBucket",
	"s3:ListBucketMultipartUploads",
	"s3:ListBucketVersions",
}

// PolicyAction reports whether a may appear in a bucket policy statement. The
// wildcard [PolicyWildcard] is not a policy action.
func PolicyAction(a string) bool {
	_, ok := slices.BinarySearch(policyActions, a)
	return ok
}

// PolicyActions returns a copy of the allowlist: every permission a bucket
// policy may grant, sorted, so that expanding [PolicyWildcard] is
// deterministic.
func PolicyActions() []string { return slices.Clone(policyActions) }

// objectAction reports whether policy action a acts on objects.
func objectAction(a string) bool { return !slices.Contains(bucketActions, a) }

// InvalidPolicyErrorName is the name of [ErrInvalidPolicy].
const InvalidPolicyErrorName = "InvalidBucketPolicy"

// ErrInvalidPolicy is returned for a policy document that is not well formed.
// The returned error wraps it with the reason.
var ErrInvalidPolicy = errors.New(InvalidPolicyErrorName, "invalid bucket policy")

// UserARN returns the ARN a document names principal id by.
func UserARN(id string) string { return UserARNPrefix + id }

// ParseUserARN returns the principal id a user ARN names. The id is whatever
// follows [UserARNPrefix]; which ids exist is the tenant's business.
func ParseUserARN(arn string) (id string, err error) {
	id, ok := strings.CutPrefix(arn, UserARNPrefix)
	if !ok {
		return "", fmt.Errorf("%q is not a user ARN %q", arn, UserARN("<id>"))
	}
	if id == "" {
		return "", fmt.Errorf("%q has an empty principal id", arn)
	}
	return id, nil
}

// BucketARN returns the ARN of bucket itself, which bucket actions need.
func BucketARN(bucket string) string { return s3ARNPrefix + bucket }

// ObjectsARN returns the ARN of every object in bucket, which object actions
// need.
func ObjectsARN(bucket string) string { return s3ARNPrefix + bucket + objectsSuffix }

// ParseS3ARN splits arn:aws:s3:::<bucket> or arn:aws:s3:::<bucket>/* into the
// bucket and whether it names the bucket's objects. Any other shape, a key
// prefix or a wildcard in the bucket name included, is an error saying why.
func ParseS3ARN(arn string) (bucket string, objects bool, err error) {
	rest, ok := strings.CutPrefix(arn, s3ARNPrefix)
	if !ok {
		return "", false, fmt.Errorf("%q is not an S3 ARN", arn)
	}
	bucket, objects = strings.CutSuffix(rest, objectsSuffix)
	if bucket == "" {
		return "", false, fmt.Errorf("%q has an empty bucket name", arn)
	}
	if strings.ContainsAny(bucket, "/*?") {
		return "", false, fmt.Errorf("%q: a bucket name must not hold /, * or ?", arn)
	}
	return bucket, objects, nil
}

// Everyone returns the principal set naming every principal of the tenant.
func Everyone() Principal { return Principal{All: true} }

// Only returns the principal set naming exactly the given ids.
func Only(ids ...string) Principal { return Principal{IDs: ids} }

// MarshalJSON encodes All as "*" and anything else as {"AWS": [user ARNs]},
// nil encoding as an empty list.
func (p Principal) MarshalJSON() ([]byte, error) {
	if p.All {
		return json.Marshal(Wildcard)
	}
	arns := make([]string, len(p.IDs))
	for i, id := range p.IDs {
		arns[i] = UserARN(id)
	}
	return json.Marshal(struct {
		AWS []string `json:"AWS"`
	}{arns})
}

// UnmarshalJSON accepts "*", {"AWS": "*"}, {"AWS": "<user ARN>"} or
// {"AWS": ["<user ARN>", ...]}, keeping each id once, in the order it first
// appears. Anything else is an error wrapping [ErrInvalidPolicy].
func (p *Principal) UnmarshalJSON(data []byte) error {
	const shape = `must be "*" or {"AWS": user ARNs}`
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("Principal: %v: %w", err, ErrInvalidPolicy)
		}
		if s != Wildcard {
			return fmt.Errorf("Principal %q: a string principal must be %q: %w", s, Wildcard, ErrInvalidPolicy)
		}
		*p = Everyone()
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil || obj == nil {
		return fmt.Errorf("Principal: %s: %w", shape, ErrInvalidPolicy)
	}
	for k := range obj {
		if k != "AWS" {
			return fmt.Errorf("Principal: unknown key %q, %s: %w", k, shape, ErrInvalidPolicy)
		}
	}
	aws, ok := obj["AWS"]
	if !ok {
		return fmt.Errorf("Principal: %s: %w", shape, ErrInvalidPolicy)
	}
	var one string
	if json.Unmarshal(aws, &one) == nil && one == Wildcard {
		*p = Everyone()
		return nil
	}
	var arns stringOrList
	if err := json.Unmarshal(aws, &arns); err != nil {
		return fmt.Errorf("Principal: %s: %w", shape, ErrInvalidPolicy)
	}
	ids := make([]string, len(arns))
	for i, arn := range arns {
		if arn == Wildcard {
			return fmt.Errorf("Principal %q must be the bare string or {\"AWS\": %q}, not a list entry: %w", Wildcard, Wildcard, ErrInvalidPolicy)
		}
		id, err := ParseUserARN(arn)
		if err != nil {
			return fmt.Errorf("Principal must be a user ARN: %v: %w", err, ErrInvalidPolicy)
		}
		ids[i] = id
	}
	*p = Principal{IDs: dedupe(ids)}
	return nil
}

// stringOrList is a JSON field AWS allows as a string or a list of strings.
type stringOrList []string

func (l *stringOrList) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("%v: %w", err, ErrInvalidPolicy)
		}
		*l = stringOrList{s}
		return nil
	}
	var ss []string
	if err := json.Unmarshal(data, &ss); err != nil {
		return fmt.Errorf("must be a string or a list of strings: %w", ErrInvalidPolicy)
	}
	*l = ss
	return nil
}

// dedupe keeps the first occurrence of each id.
func dedupe(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	out := ids[:0]
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// UnmarshalJSON parses a policy document strictly and refuses one that is not
// well formed, so an invalid policy never comes into existence: a field the
// schema does not define, a property name that differs from the schema in
// case, a malformed principal, or a document [WellFormed] rejects is an error
// wrapping [ErrInvalidPolicy]. Lists keep the order the document gives them.
func (p *BucketPolicy) UnmarshalJSON(data []byte) error {
	if err := strictKeys(data); err != nil {
		return err
	}
	// The wire form takes Action and Resource as a string or a list.
	var w struct {
		Version   string
		Statement []struct {
			Sid       string
			Effect    Effect
			Principal Principal
			Action    stringOrList
			Resource  stringOrList
		}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&w); err != nil {
		if errors.Is(err, ErrInvalidPolicy) {
			return fmt.Errorf("decoding policy: %w", err)
		}
		return fmt.Errorf("decoding policy: %v: %w", err, ErrInvalidPolicy)
	}
	d := BucketPolicy{Version: w.Version, Statements: make([]Statement, len(w.Statement))}
	for i, st := range w.Statement {
		d.Statements[i] = Statement{Sid: st.Sid, Effect: st.Effect, Principal: st.Principal, Actions: st.Action, Resources: st.Resource}
	}
	if err := WellFormed(d); err != nil {
		return err
	}
	*p = d
	return nil
}

// strictKeys rejects any property name that is not spelled exactly as the
// schema defines it, and a null Sid. encoding/json matches field names
// case-insensitively, so DisallowUnknownFields alone would accept "statement",
// "effect" or "ACTION".
// A document this cannot parse is left to the decoder, which reports it.
func strictKeys(data []byte) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return nil
	}
	for k := range top {
		if k != "Version" && k != "Statement" {
			return fmt.Errorf("decoding policy: unknown field %q: %w", k, ErrInvalidPolicy)
		}
	}
	var statements []map[string]json.RawMessage
	if err := json.Unmarshal(top["Statement"], &statements); err != nil {
		return nil
	}
	for _, st := range statements {
		for k, v := range st {
			switch k {
			case "Sid":
				// encoding/json decodes null into "" without error, which would
				// read as an absent Sid.
				if string(bytes.TrimSpace(v)) == "null" {
					return fmt.Errorf("decoding policy: Sid must be a string: %w", ErrInvalidPolicy)
				}
			case "Effect", "Principal", "Action", "Resource":
			default:
				return fmt.Errorf("decoding policy: unknown field %q: %w", k, ErrInvalidPolicy)
			}
		}
	}
	return nil
}

// WellFormed is the structural check every decoded document passes: a
// recognized Version, at least one statement and at most [MaxSetSize], each
// with a recognized effect, naming [Wildcard] or at least one principal and
// at most MaxSetSize (and never Wildcard inside the list), holding at least
// one action, each a [PolicyAction] or [PolicyWildcard], and at least one
// resource, each an ARN of the one bucket every resource in the document
// names, covering what its actions act on: an object action needs the objects
// ARN in its statement, a bucket action the bucket ARN, and "s3:*" both. It
// returns an error wrapping [ErrInvalidPolicy] otherwise.
func WellFormed(d BucketPolicy) error {
	if d.Version != Version2012 && d.Version != Version2008 {
		return fmt.Errorf("policy Version %q must be %q or %q: %w", d.Version, Version2012, Version2008, ErrInvalidPolicy)
	}
	if len(d.Statements) == 0 {
		return fmt.Errorf("Statement must not be empty: %w", ErrInvalidPolicy)
	}
	if len(d.Statements) > MaxSetSize {
		return fmt.Errorf("Statement holds %d statements, at most %d are allowed: %w", len(d.Statements), MaxSetSize, ErrInvalidPolicy)
	}
	bucket := ""
	for i, st := range d.Statements {
		if st.Effect != Allow && st.Effect != Deny {
			return fmt.Errorf("statement %d: Effect %q must be %q or %q: %w", i, st.Effect, Allow, Deny, ErrInvalidPolicy)
		}
		if !st.Principal.All {
			if len(st.Principal.IDs) == 0 {
				return fmt.Errorf("statement %d: Principal must not be empty: %w", i, ErrInvalidPolicy)
			}
			if len(st.Principal.IDs) > MaxSetSize {
				return fmt.Errorf("statement %d: Principal names %d principals, at most %d are allowed: %w", i, len(st.Principal.IDs), MaxSetSize, ErrInvalidPolicy)
			}
			if slices.Contains(st.Principal.IDs, Wildcard) {
				return fmt.Errorf("statement %d: Principal %q must be the bare string, not a list entry: %w", i, Wildcard, ErrInvalidPolicy)
			}
		}
		if len(st.Actions) == 0 {
			return fmt.Errorf("statement %d: Action must not be empty: %w", i, ErrInvalidPolicy)
		}
		for _, a := range st.Actions {
			if a != PolicyWildcard && !PolicyAction(a) {
				return fmt.Errorf("statement %d: Action %q is not a policy action: %w", i, a, ErrInvalidPolicy)
			}
		}
		if len(st.Resources) == 0 {
			return fmt.Errorf("statement %d: Resource must not be empty: %w", i, ErrInvalidPolicy)
		}
		var hasBucket, hasObjects bool
		for _, r := range st.Resources {
			b, objects, err := ParseS3ARN(r)
			if err != nil {
				return fmt.Errorf("statement %d: Resource %q must be %q or %q: %v: %w", i, r, BucketARN("<bucket>"), ObjectsARN("<bucket>"), err, ErrInvalidPolicy)
			}
			if bucket == "" {
				bucket = b
			}
			if b != bucket {
				return fmt.Errorf("statement %d: Resource %q names bucket %q, the policy names %q: %w", i, r, b, bucket, ErrInvalidPolicy)
			}
			hasBucket = hasBucket || !objects
			hasObjects = hasObjects || objects
		}
		for _, a := range st.Actions {
			wildcard := a == PolicyWildcard
			if (wildcard || objectAction(a)) && !hasObjects {
				return fmt.Errorf("statement %d: Action %q needs Resource %q: %w", i, a, ObjectsARN(bucket), ErrInvalidPolicy)
			}
			if (wildcard || !objectAction(a)) && !hasBucket {
				return fmt.Errorf("statement %d: Action %q needs Resource %q: %w", i, a, BucketARN(bucket), ErrInvalidPolicy)
			}
		}
	}
	return nil
}
