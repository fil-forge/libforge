package bucket_test

import (
	"bytes"
	"encoding/json"
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
					Actions:   []string{"s3:GetObject"},
					Resources: []string{"arn:aws:s3:::b/*"},
				},
				{
					Effect:    bucket.Deny,
					Principal: bucket.Everyone(),
					Actions:   []string{"s3:DeleteObject"},
					Resources: []string{"arn:aws:s3:::b/*"},
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
}
