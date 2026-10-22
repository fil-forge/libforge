package bucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/fil-forge/libforge/commands"
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
// policy operation could rewrite the policy that granted it, a key that
// rewrites the bucket's configuration (versioning, default object lock,
// tags, encryption) changes it for every principal, and every principal
// holds s3:ListAllMyBuckets. Reading that configuration is a policy action.
var policyActions = sorted([]string{
	"s3:AbortMultipartUpload",
	"s3:BypassGovernanceRetention",
	"s3:DeleteObject",
	"s3:DeleteObjectTagging",
	"s3:DeleteObjectVersion",
	"s3:GetBucketAcl",
	"s3:GetBucketCORS",
	"s3:GetBucketLocation",
	"s3:GetBucketObjectLockConfiguration",
	"s3:GetBucketTagging",
	"s3:GetBucketVersioning",
	"s3:GetObject",
	"s3:GetObjectAttributes",
	"s3:GetObjectLegalHold",
	"s3:GetObjectRetention",
	"s3:GetObjectTagging",
	"s3:GetObjectVersion",
	"s3:ListBucket",
	"s3:ListBucketMultipartUploads",
	"s3:ListBucketVersions",
	"s3:ListMultipartUploadParts",
	"s3:PutObject",
	"s3:PutObjectLegalHold",
	"s3:PutObjectRetention",
	"s3:PutObjectTagging",
})

func sorted(s []string) []string {
	slices.Sort(s)
	return s
}

// bucketActions are the policy actions that act on the bucket rather than on
// its objects; every other policy action is an object action. The split is
// versitygw's, which evaluates the policy against these ARNs.
var bucketActions = []string{
	"s3:GetBucketAcl",
	"s3:GetBucketCORS",
	"s3:GetBucketLocation",
	"s3:GetBucketObjectLockConfiguration",
	"s3:GetBucketTagging",
	"s3:GetBucketVersioning",
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

// NewStringSet returns the set of the given members.
func NewStringSet(members ...string) StringSet {
	s := make(StringSet, len(members))
	for _, m := range members {
		s[m] = commands.Unit{}
	}
	return s
}

// Members returns the set's members, sorted, never nil.
func (s StringSet) Members() []string {
	out := make([]string, 0, len(s))
	for m := range s {
		out = append(out, m)
	}
	slices.Sort(out)
	return out
}

// MarshalJSON encodes the set as the sorted list of its members; nil encodes
// as an empty list.
func (s StringSet) MarshalJSON() ([]byte, error) { return json.Marshal(s.Members()) }

// UnmarshalJSON accepts a string or a list of strings, as AWS allows for
// Action and Resource, holding at most [MaxSetSize] entries; duplicates
// collapse. Anything else is an error wrapping [ErrInvalidPolicy].
func (s *StringSet) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	var members []string
	if len(data) > 0 && data[0] == '"' {
		var one string
		if err := json.Unmarshal(data, &one); err != nil {
			return fmt.Errorf("%v: %w", err, ErrInvalidPolicy)
		}
		members = []string{one}
	} else if err := json.Unmarshal(data, &members); err != nil {
		return fmt.Errorf("must be a string or a list of strings: %w", ErrInvalidPolicy)
	}
	if len(members) > MaxSetSize {
		return fmt.Errorf("holds %d entries, at most %d are allowed: %w", len(members), MaxSetSize, ErrInvalidPolicy)
	}
	*s = NewStringSet(members...)
	return nil
}

// Everyone returns the principal set naming every principal of the tenant.
func Everyone() Principal { return Principal{Wildcard: {}} }

// Only returns the principal set naming exactly the given ids.
func Only(ids ...string) Principal { return Principal(NewStringSet(ids...)) }

// All reports whether p names every principal of the tenant: it holds
// [Wildcard].
func (p Principal) All() bool {
	_, ok := p[Wildcard]
	return ok
}

// Members returns p's members, sorted, never nil.
func (p Principal) Members() []string { return StringSet(p).Members() }

// MarshalJSON encodes a set holding [Wildcard] as "*" and anything else as
// {"AWS": [user ARNs]}, sorted, nil encoding as an empty list.
func (p Principal) MarshalJSON() ([]byte, error) {
	if p.All() {
		return json.Marshal(Wildcard)
	}
	ids := p.Members()
	arns := make([]string, len(ids))
	for i, id := range ids {
		arns[i] = UserARN(id)
	}
	return json.Marshal(struct {
		AWS []string `json:"AWS"`
	}{arns})
}

// UnmarshalJSON accepts "*", {"AWS": "*"}, {"AWS": "<user ARN>"} or
// {"AWS": ["<user ARN>", ...]}. Anything else is an error wrapping
// [ErrInvalidPolicy].
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
	var arns StringSet
	if err := json.Unmarshal(aws, &arns); err != nil {
		return fmt.Errorf("Principal: %s: %w", shape, ErrInvalidPolicy)
	}
	ids := make(Principal, len(arns))
	for _, arn := range arns.Members() {
		if arn == Wildcard {
			return fmt.Errorf("Principal %q must be the bare string or {\"AWS\": %q}, not a list entry: %w", Wildcard, Wildcard, ErrInvalidPolicy)
		}
		id, err := ParseUserARN(arn)
		if err != nil {
			return fmt.Errorf("Principal must be a user ARN: %v: %w", err, ErrInvalidPolicy)
		}
		ids[id] = commands.Unit{}
	}
	*p = ids
	return nil
}

// UnmarshalJSON parses a policy document strictly and refuses one that is not
// well formed, so an invalid policy never comes into existence: a field the
// schema does not define, a property name that differs from the schema in
// case, a malformed principal, or a document [WellFormed] rejects is an error
// wrapping [ErrInvalidPolicy]. Action and Resource may each be a string or a
// list; the lists are sets, so order and duplicates do not survive.
func (p *PolicyDocument) UnmarshalJSON(data []byte) error {
	if err := strictKeys(data); err != nil {
		return err
	}
	type plain PolicyDocument // the struct without this method
	var d plain
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		if errors.Is(err, ErrInvalidPolicy) {
			return fmt.Errorf("decoding policy: %w", err)
		}
		return fmt.Errorf("decoding policy: %v: %w", err, ErrInvalidPolicy)
	}
	if err := WellFormed(PolicyDocument(d)); err != nil {
		return err
	}
	*p = PolicyDocument(d)
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
func WellFormed(d PolicyDocument) error {
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
		switch {
		case st.Principal.All() && len(st.Principal) > 1:
			return fmt.Errorf("statement %d: Principal %q must be the bare string, not a list entry: %w", i, Wildcard, ErrInvalidPolicy)
		case len(st.Principal) == 0:
			return fmt.Errorf("statement %d: Principal must not be empty: %w", i, ErrInvalidPolicy)
		case len(st.Principal) > MaxSetSize:
			return fmt.Errorf("statement %d: Principal names %d principals, at most %d are allowed: %w", i, len(st.Principal), MaxSetSize, ErrInvalidPolicy)
		}
		if len(st.Actions) == 0 {
			return fmt.Errorf("statement %d: Action must not be empty: %w", i, ErrInvalidPolicy)
		}
		for _, a := range st.Actions.Members() {
			if a != PolicyWildcard && !PolicyAction(a) {
				return fmt.Errorf("statement %d: Action %q is not a policy action: %w", i, a, ErrInvalidPolicy)
			}
		}
		if len(st.Resources) == 0 {
			return fmt.Errorf("statement %d: Resource must not be empty: %w", i, ErrInvalidPolicy)
		}
		var hasBucket, hasObjects bool
		for _, r := range st.Resources.Members() {
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
		for _, a := range st.Actions.Members() {
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
