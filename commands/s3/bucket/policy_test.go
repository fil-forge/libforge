package bucket_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fil-forge/libforge/commands/s3/bucket"
	"github.com/stretchr/testify/require"
)

func TestPolicyOKRoundTrip(t *testing.T) {
	in := &bucket.PolicyOK{
		ETag: `"bafy"`,
		Policy: &bucket.BucketPolicy{
			Version: "2012-10-17",
			Statements: []bucket.Statement{
				{
					Sid:       "readers",
					Effect:    bucket.Allow,
					Principal: bucket.Only("alice", "bob"),
					Actions:   bucket.NewStringSet("s3:GetObject"),
					Resources: bucket.NewStringSet("arn:aws:s3:::b/*"),
				},
				{
					Effect:    bucket.Deny,
					Principal: bucket.Everyone(),
					Actions:   bucket.NewStringSet("s3:DeleteObject"),
					Resources: bucket.NewStringSet("arn:aws:s3:::b/*"),
				},
			},
		},
	}

	var cb bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&cb))
	var out bucket.PolicyOK
	require.NoError(t, out.UnmarshalCBOR(bytes.NewReader(cb.Bytes())))
	require.Equal(t, *in, out)

	var jb bytes.Buffer
	require.NoError(t, in.MarshalDagJSON(&jb))
	var outJSON bucket.PolicyOK
	require.NoError(t, outJSON.UnmarshalDagJSON(bytes.NewReader(jb.Bytes())))
	require.Equal(t, *in, outJSON)

	aws, err := json.Marshal(out.Policy)
	require.NoError(t, err)
	require.JSONEq(t, `{"Version":"2012-10-17","Statement":[`+
		`{"Sid":"readers","Effect":"Allow","Principal":{"AWS":["arn:aws:iam:::user/alice","arn:aws:iam:::user/bob"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::b/*"]},`+
		`{"Effect":"Deny","Principal":"*","Action":["s3:DeleteObject"],"Resource":["arn:aws:s3:::b/*"]}]}`, string(aws))
}

func TestPolicyOKWithoutPolicy(t *testing.T) {
	in := &bucket.PolicyOK{ETag: `"bafy"`}
	var cb bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&cb))
	var out bucket.PolicyOK
	require.NoError(t, out.UnmarshalCBOR(bytes.NewReader(cb.Bytes())))
	require.Nil(t, out.Policy)
}

func TestPrincipalJSON(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bucket.Principal
	}{
		{`"*"`, bucket.Everyone()},
		{`{"AWS":"*"}`, bucket.Everyone()},
		{`{"AWS":"arn:aws:iam:::user/alice"}`, bucket.Only("alice")},
		{`{"AWS":["arn:aws:iam:::user/alice","arn:aws:iam:::user/alice"]}`, bucket.Only("alice")},
	} {
		var p bucket.Principal
		require.NoError(t, json.Unmarshal([]byte(tc.in), &p), tc.in)
		require.Equal(t, tc.want, p, tc.in)
	}
	for _, bad := range []string{`"alice"`, `{"AWS":["*"]}`, `{"AWS":"alice"}`, `{"Service":"x"}`, `{}`, `null`} {
		var p bucket.Principal
		require.Error(t, json.Unmarshal([]byte(bad), &p), bad)
	}

	t.Run("an id round-trips byte for byte, nothing is escaped", func(t *testing.T) {
		in := bucket.Only(`a/b:c"d`)
		raw, err := json.Marshal(in)
		require.NoError(t, err)
		var back bucket.Principal
		require.NoError(t, json.Unmarshal(raw, &back))
		require.Equal(t, in, back)
	})
}

func TestParseS3ARN(t *testing.T) {
	b, objects, err := bucket.ParseS3ARN("arn:aws:s3:::photos")
	require.NoError(t, err)
	require.Equal(t, "photos", b)
	require.False(t, objects)

	b, objects, err = bucket.ParseS3ARN("arn:aws:s3:::photos/*")
	require.NoError(t, err)
	require.Equal(t, "photos", b)
	require.True(t, objects)

	for _, tc := range []struct{ arn, want string }{
		{"photos", "not an S3 ARN"},
		{"arn:aws:iam:::user/alice", "not an S3 ARN"},
		{"arn:aws:s3:::", "empty bucket name"},
		{"arn:aws:s3:::/*", "empty bucket name"},
		{"arn:aws:s3:::photos/2024/*", "bucket name must not hold"},
		{"arn:aws:s3:::photos/cat.jpg", "bucket name must not hold"},
		{"arn:aws:s3:::*/*", "bucket name must not hold"},
		{"arn:aws:s3:::ph?tos", "bucket name must not hold"},
	} {
		_, _, err := bucket.ParseS3ARN(tc.arn)
		require.ErrorContains(t, err, tc.want, tc.arn)
	}
}

func TestParseUserARN(t *testing.T) {
	id, err := bucket.ParseUserARN(bucket.UserARN("alice"))
	require.NoError(t, err)
	require.Equal(t, "alice", id)

	for _, tc := range []struct{ arn, want string }{
		{"alice", "not a user ARN"},
		{"arn:aws:iam::123456789012:user/alice", "not a user ARN"},
		{"arn:aws:iam:::role/alice", "not a user ARN"},
		{"arn:aws:iam:::user/", "empty principal id"},
	} {
		_, err := bucket.ParseUserARN(tc.arn)
		require.ErrorContains(t, err, tc.want, tc.arn)
	}
}

// policyJSON wraps statements, given as JSON, in a 2012-10-17 document.
func policyJSON(statements ...string) string {
	return `{"Version": "2012-10-17", "Statement": [` + strings.Join(statements, ",") + `]}`
}

// res is the Resource member covering every policy action on photos.
const res = `"Resource": ["arn:aws:s3:::photos", "arn:aws:s3:::photos/*"]`

func TestBucketPolicyUnmarshalJSON(t *testing.T) {
	t.Run("decodes the documented shape, keeping its order", func(t *testing.T) {
		var d bucket.BucketPolicy
		require.NoError(t, json.Unmarshal([]byte(policyJSON(
			`{"Sid": "owners", "Effect": "Allow", "Principal": {"AWS": ["arn:aws:iam:::user/bob", "arn:aws:iam:::user/alice"]}, "Action": ["s3:ListBucket", "s3:GetObject"], `+res+`}`,
			`{"Effect": "Deny", "Principal": "*", "Action": "s3:PutObjectRetention", "Resource": "arn:aws:s3:::photos/*"}`,
		)), &d))
		require.Equal(t, bucket.BucketPolicy{Version: bucket.Version2012, Statements: []bucket.Statement{
			{Sid: "owners", Effect: bucket.Allow, Principal: bucket.Only("bob", "alice"), Actions: bucket.NewStringSet("s3:ListBucket", "s3:GetObject"), Resources: bucket.NewStringSet(bucket.BucketARN("photos"), bucket.ObjectsARN("photos"))},
			{Effect: bucket.Deny, Principal: bucket.Everyone(), Actions: bucket.NewStringSet("s3:PutObjectRetention"), Resources: bucket.NewStringSet(bucket.ObjectsARN("photos"))},
		}}, d)
	})

	rejects := []struct {
		name string
		body string
		want string
	}{
		{"a missing version", `{"Statement": [{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], ` + res + `}]}`, `Version "" must be`},
		{"an unknown version", `{"Version": "2024-01-01", "Statement": [{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], ` + res + `}]}`, `Version "2024-01-01" must be`},
		{"unknown top-level field", `{"Version": "2012-10-17", "Statement": [], "Id": "x"}`, `unknown field "Id"`},
		{"a condition", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], ` + res + `, "Condition": {}}`), `unknown field "Condition"`},
		{"a not-principal", policyJSON(`{"Effect": "Deny", "NotPrincipal": "*", "Action": ["s3:GetObject"], ` + res + `}`), `unknown field "NotPrincipal"`},
		{"old field names", `{"statements": [{"effect": "allow", "principals": ["alice"], "actions": ["s3:GetObject"]}]}`, `unknown field "statements"`},
		{"a string principal other than the wildcard", policyJSON(`{"Effect": "Allow", "Principal": "alice", "Action": ["s3:GetObject"], ` + res + `}`), `a string principal must be "*"`},
		{"a bare id list principal", policyJSON(`{"Effect": "Allow", "Principal": ["alice"], "Action": ["s3:GetObject"], ` + res + `}`), `must be "*" or {"AWS": user ARNs}`},
		{"an ARN with an account", policyJSON(`{"Effect": "Allow", "Principal": {"AWS": "arn:aws:iam::123456789012:user/alice"}, "Action": ["s3:GetObject"], ` + res + `}`), `must be a user ARN`},
		{"a bare id inside AWS", policyJSON(`{"Effect": "Allow", "Principal": {"AWS": ["alice"]}, "Action": ["s3:GetObject"], ` + res + `}`), `must be a user ARN`},
		{"a user ARN without an id", policyJSON(`{"Effect": "Allow", "Principal": {"AWS": "arn:aws:iam:::user/"}, "Action": ["s3:GetObject"], ` + res + `}`), `must be a user ARN`},
		{"a role ARN", policyJSON(`{"Effect": "Allow", "Principal": {"AWS": "arn:aws:iam:::role/alice"}, "Action": ["s3:GetObject"], ` + res + `}`), `must be a user ARN`},
		{"a principal key other than AWS", policyJSON(`{"Effect": "Allow", "Principal": {"Service": "s3.amazonaws.com"}, "Action": ["s3:GetObject"], ` + res + `}`), `unknown key "Service"`},
		{"a lowercase AWS key", policyJSON(`{"Effect": "Allow", "Principal": {"aws": "*"}, "Action": ["s3:GetObject"], ` + res + `}`), `unknown key "aws"`},
		{"AWS beside another key", policyJSON(`{"Effect": "Allow", "Principal": {"AWS": "*", "Federated": "x"}, "Action": ["s3:GetObject"], ` + res + `}`), `unknown key "Federated"`},
		{"an empty principal object", policyJSON(`{"Effect": "Allow", "Principal": {}, "Action": ["s3:GetObject"], ` + res + `}`), `must be "*" or {"AWS": user ARNs}`},
		{"the wildcard inside the AWS list", policyJSON(`{"Effect": "Allow", "Principal": {"AWS": ["arn:aws:iam:::user/alice", "*"]}, "Action": ["s3:GetObject"], ` + res + `}`), `not a list entry`},
		{"a lowercase top-level field", `{"Version": "2012-10-17", "statement": []}`, `unknown field "statement"`},
		{"a lowercase version field", `{"version": "2012-10-17", "Statement": []}`, `unknown field "version"`},
		{"a lowercase statement field", policyJSON(`{"effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], ` + res + `}`), `unknown field "effect"`},
		{"an upper-case statement field", policyJSON(`{"Effect": "Allow", "Principal": "*", "ACTION": ["s3:GetObject"], ` + res + `}`), `unknown field "ACTION"`},
		{"a lowercase resource field", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], "resource": "arn:aws:s3:::photos/*"}`), `unknown field "resource"`},
		{"a null sid", policyJSON(`{"Sid": null, "Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], ` + res + `}`), `Sid must be a string`},
		{"an action that is neither string nor list", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": 7, ` + res + `}`), `must be a string or a list of strings`},
		// The structure is checked while parsing, so an invalid policy never
		// comes into existence.
		{"empty statements", `{"Version": "2012-10-17", "Statement": []}`, `Statement must not be empty`},
		{"an unknown effect", policyJSON(`{"Effect": "Permit", "Principal": "*", "Action": ["s3:GetObject"], ` + res + `}`), `Effect "Permit"`},
		{"a lowercase effect", policyJSON(`{"Effect": "allow", "Principal": "*", "Action": ["s3:GetObject"], ` + res + `}`), `Effect "allow"`},
		{"an empty principal list", policyJSON(`{"Effect": "Allow", "Principal": {"AWS": []}, "Action": ["s3:GetObject"], ` + res + `}`), `Principal must not be empty`},
		{"a missing principal", policyJSON(`{"Effect": "Allow", "Action": ["s3:GetObject"], ` + res + `}`), `Principal must not be empty`},
		{"empty actions", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": [], ` + res + `}`), `Action must not be empty`},
		{"a bucket-level action", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:CreateBucket"], ` + res + `}`), `Action "s3:CreateBucket"`},
		{"an unknown action", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:Frobnicate"], ` + res + `}`), `Action "s3:Frobnicate"`},
		{"a bare star action", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["*"], ` + res + `}`), `Action "*"`},
		{"an invalid action in a later statement", policyJSON(
			`{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], `+res+`}`,
			`{"Effect": "Deny", "Principal": {"AWS": "arn:aws:iam:::user/alice"}, "Action": ["s3:ListAllMyBuckets"], `+res+`}`,
		), `statement 1: Action "s3:ListAllMyBuckets"`},
		{"a missing resource", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"]}`), `Resource must not be empty`},
		{"an empty resource list", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], "Resource": []}`), `Resource must not be empty`},
		{"a resource that is not an S3 ARN", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], "Resource": "photos"}`), `Resource "photos" must be`},
		{"a key-prefix resource", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], "Resource": "arn:aws:s3:::photos/2024/*"}`), `Resource "arn:aws:s3:::photos/2024/*" must be`},
		{"a single-object resource", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], "Resource": "arn:aws:s3:::photos/cat.jpg"}`), `must be`},
		{"a wildcard bucket", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], "Resource": "arn:aws:s3:::*/*"}`), `must be`},
		{"an empty bucket", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:ListBucket"], "Resource": "arn:aws:s3:::"}`), `must be`},
		{"resources naming two buckets", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], "Resource": ["arn:aws:s3:::photos/*", "arn:aws:s3:::videos/*"]}`), `names bucket "videos", the policy names "photos"`},
		{"statements naming two buckets", policyJSON(
			`{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], "Resource": "arn:aws:s3:::photos/*"}`,
			`{"Effect": "Allow", "Principal": "*", "Action": ["s3:ListBucket"], "Resource": "arn:aws:s3:::videos"}`,
		), `statement 1: Resource "arn:aws:s3:::videos" names bucket "videos"`},
		{"an object action on the bucket ARN", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:GetObject"], "Resource": "arn:aws:s3:::photos"}`), `Action "s3:GetObject" needs Resource "arn:aws:s3:::photos/*"`},
		{"a bucket action on the objects ARN", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": ["s3:ListBucket"], "Resource": "arn:aws:s3:::photos/*"}`), `Action "s3:ListBucket" needs Resource "arn:aws:s3:::photos"`},
		{"the action wildcard on the objects ARN alone", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": "s3:*", "Resource": "arn:aws:s3:::photos/*"}`), `Action "s3:*" needs Resource "arn:aws:s3:::photos"`},
		{"the action wildcard on the bucket ARN alone", policyJSON(`{"Effect": "Allow", "Principal": "*", "Action": "s3:*", "Resource": "arn:aws:s3:::photos"}`), `Action "s3:*" needs Resource "arn:aws:s3:::photos/*"`},
	}
	for _, tt := range rejects {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			var d bucket.BucketPolicy
			err := json.Unmarshal([]byte(tt.body), &d)
			require.ErrorIs(t, err, bucket.ErrInvalidPolicy)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestPolicyActions(t *testing.T) {
	actions := bucket.PolicyActions()
	require.IsIncreasing(t, actions)
	for _, a := range actions {
		require.True(t, bucket.PolicyAction(a), a)
	}
	for _, a := range []string{bucket.PolicyWildcard, "*", "s3:CreateBucket", "s3:DeleteBucket", "s3:ListAllMyBuckets", "s3:PutBucketPolicy"} {
		require.False(t, bucket.PolicyAction(a), a)
	}
}

func TestStatementIPLD(t *testing.T) {
	st := bucket.Statement{
		Effect:    bucket.Allow,
		Principal: bucket.Only("bob", "alice", "bob"),
		Actions:   bucket.NewStringSet("s3:PutObject", "s3:GetObject", "s3:GetObject"),
		Resources: bucket.NewStringSet("arn:aws:s3:::b/*", "arn:aws:s3:::b"),
	}

	t.Run("DAG-JSON is the sets form with sorted keys", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, st.MarshalDagJSON(&buf))
		require.Equal(t, `{"action":{"s3:GetObject":{},"s3:PutObject":{}},"effect":"Allow","principal":{"alice":{},"bob":{}},"resource":{"arn:aws:s3:::b":{},"arn:aws:s3:::b/*":{}}}`, buf.String())

		withSid := st
		withSid.Sid = "rw"
		buf.Reset()
		require.NoError(t, withSid.MarshalDagJSON(&buf))
		require.Contains(t, buf.String(), `,"sid":"rw"}`)
	})

	t.Run("the wildcard principal is the single member *", func(t *testing.T) {
		all := st
		all.Principal = bucket.Everyone()
		var buf bytes.Buffer
		require.NoError(t, all.MarshalDagJSON(&buf))
		require.Contains(t, buf.String(), `"principal":{"*":{}}`)
	})

	t.Run("decoding round-trips, sets never nil", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, st.MarshalDagJSON(&buf))
		var back bucket.Statement
		require.NoError(t, back.UnmarshalDagJSON(bytes.NewReader(buf.Bytes())))
		require.Equal(t, st, back)

		var empty bucket.Statement
		require.NoError(t, empty.UnmarshalDagJSON(strings.NewReader(`{"action":{},"effect":"Deny","principal":{},"resource":{}}`)))
		require.Equal(t, bucket.Statement{Effect: bucket.Deny, Principal: bucket.Principal{}, Actions: bucket.StringSet{}, Resources: bucket.StringSet{}}, empty)
	})

	t.Run("CBOR round trip", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, st.MarshalCBOR(&buf))
		var back bucket.Statement
		require.NoError(t, back.UnmarshalCBOR(bytes.NewReader(buf.Bytes())))
		require.Equal(t, st, back)
	})
}

func TestStringSetJSON(t *testing.T) {
	t.Run("marshals as the sorted list, nil as an empty list", func(t *testing.T) {
		raw, err := json.Marshal(bucket.NewStringSet("b", "a", "b"))
		require.NoError(t, err)
		require.Equal(t, `["a","b"]`, string(raw))

		raw, err = json.Marshal(bucket.Statement{})
		require.NoError(t, err)
		require.Equal(t, `{"Effect":"","Principal":{"AWS":[]},"Action":[],"Resource":[]}`, string(raw))
	})

	t.Run("unmarshals a string or a list", func(t *testing.T) {
		var s bucket.StringSet
		require.NoError(t, json.Unmarshal([]byte(`"x"`), &s))
		require.Equal(t, bucket.NewStringSet("x"), s)
		require.NoError(t, json.Unmarshal([]byte(`["y","x","y"]`), &s))
		require.Equal(t, bucket.NewStringSet("x", "y"), s)
		for _, bad := range []string{`1`, `{"x":1}`, `[1]`} {
			require.ErrorIs(t, json.Unmarshal([]byte(bad), &s), bucket.ErrInvalidPolicy, bad)
		}
	})
}
