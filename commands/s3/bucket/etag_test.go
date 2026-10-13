package bucket_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/fil-forge/libforge/commands/s3/bucket"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
)

const (
	bucketARN  = "arn:aws:s3:::photos"
	objectsARN = "arn:aws:s3:::photos/*"
)

var everyone = bucket.Everyone()

func only(ids ...string) bucket.Principal { return bucket.Only(ids...) }

// both is the resource set every helper statement carries.
func both() bucket.StringSet { return bucket.NewStringSet(bucketARN, objectsARN) }

func allow(p bucket.Principal, actions ...string) bucket.Statement {
	return bucket.Statement{Effect: bucket.Allow, Principal: p, Actions: bucket.NewStringSet(actions...), Resources: both()}
}

func deny(p bucket.Principal, actions ...string) bucket.Statement {
	return bucket.Statement{Effect: bucket.Deny, Principal: p, Actions: bucket.NewStringSet(actions...), Resources: both()}
}

func doc(statements ...bucket.Statement) bucket.BucketPolicy {
	return bucket.BucketPolicy{Version: bucket.Version2012, Statements: statements}
}

func TestNormalize(t *testing.T) {
	t.Run("orders statements by CID and drops duplicates", func(t *testing.T) {
		a := doc(allow(only("alice"), "s3:GetObject"), deny(everyone, "s3:DeleteObject"))
		b := doc(deny(everyone, "s3:DeleteObject"), allow(only("alice"), "s3:GetObject"), deny(everyone, "s3:DeleteObject"))
		require.Equal(t, bucket.Normalize(a), bucket.Normalize(b))
		require.Len(t, bucket.Normalize(b).Statements, 2)
		keys := []string{bucket.StatementCID(a.Statements[0]).String(), bucket.StatementCID(a.Statements[1]).String()}
		require.True(t, slices.IsSorted(keys) || !slices.IsSorted(keys)) // either order is fine in a; the output is sorted:
		n := bucket.Normalize(b)
		require.True(t, bucket.StatementCID(n.Statements[0]).String() < bucket.StatementCID(n.Statements[1]).String())
	})

	t.Run("keeps the version and does not alias the input", func(t *testing.T) {
		d := doc(allow(only("alice"), "s3:GetObject"))
		n := bucket.Normalize(d)
		require.Equal(t, bucket.Version2012, n.Version)
		n.Statements[0] = deny(everyone, "s3:GetObject")
		require.Equal(t, bucket.Allow, d.Statements[0].Effect)
	})

	t.Run("nil and empty statement lists normalize the same", func(t *testing.T) {
		require.Equal(t, bucket.Normalize(bucket.BucketPolicy{}), bucket.Normalize(bucket.BucketPolicy{Statements: []bucket.Statement{}}))
	})
}

func TestETag(t *testing.T) {
	d := doc(allow(only("alice"), "s3:GetObject"))

	t.Run("is the quoted dag-json CID of the normalized document", func(t *testing.T) {
		tag := bucket.ETag(d)
		require.Regexp(t, `^"bagu[a-z2-7]+"$`, tag)
		c, err := cid.Decode(tag[1 : len(tag)-1])
		require.NoError(t, err)
		require.EqualValues(t, cid.DagJSON, c.Type())
		statement := `{"action":{"s3:GetObject":{}},"effect":"Allow","principal":{"alice":{}},"resource":{"arn:aws:s3:::photos":{},"arn:aws:s3:::photos/*":{}}}`
		require.Equal(t, dagJSONCID(t, dagJSONPolicy(t, "2012-10-17", statement)), c)
		require.Equal(t, bucket.ETag(bucket.Normalize(d)), tag)
	})

	t.Run("encodes a sid, a wildcard principal and several statements", func(t *testing.T) {
		d := doc(
			bucket.Statement{Sid: "owners", Effect: bucket.Allow, Principal: only("bob", "alice"), Actions: bucket.NewStringSet("s3:ListBucket", "s3:GetObject"), Resources: both()},
			bucket.Statement{Effect: bucket.Deny, Principal: everyone, Actions: bucket.NewStringSet("s3:PutObjectRetention"), Resources: bucket.NewStringSet(objectsARN)},
		)
		want := dagJSONPolicy(t, "2012-10-17",
			dagJSONStatement("owners", bucket.Allow, []string{"alice", "bob"}, []string{"s3:GetObject", "s3:ListBucket"}, both().Members()),
			dagJSONStatement("", bucket.Deny, []string{"*"}, []string{"s3:PutObjectRetention"}, []string{objectsARN}),
		)
		require.Equal(t, `"`+dagJSONCID(t, want).String()+`"`, bucket.ETag(d))
	})

	t.Run("order and duplicates are not significant", func(t *testing.T) {
		a := doc(allow(only("alice", "bob"), "s3:GetObject", "s3:PutObject"), deny(everyone, "s3:DeleteObject"))
		dup := deny(everyone, "s3:DeleteObject")
		dup.Resources = bucket.NewStringSet(objectsARN, bucketARN, bucketARN)
		b := doc(
			deny(everyone, "s3:DeleteObject"),
			allow(only("bob", "alice", "bob"), "s3:PutObject", "s3:GetObject", "s3:PutObject"),
			dup,
		)
		require.Equal(t, bucket.ETag(a), bucket.ETag(b))
	})

	t.Run("is stable and changes with the document", func(t *testing.T) {
		require.Equal(t, bucket.ETag(d), bucket.ETag(doc(allow(only("alice"), "s3:GetObject"))))
		require.NotEqual(t, bucket.ETag(d), bucket.ETag(doc(allow(only("alice"), "s3:PutObject"))))
		require.NotEqual(t, bucket.ETag(d), bucket.ETag(doc(allow(only("bob"), "s3:GetObject"))))
		require.NotEqual(t, bucket.ETag(d), bucket.ETag(doc(deny(only("alice"), "s3:GetObject"))))
		withSid := doc(bucket.Statement{Sid: "read", Effect: bucket.Allow, Principal: only("alice"), Actions: bucket.NewStringSet("s3:GetObject"), Resources: both()})
		require.NotEqual(t, bucket.ETag(d), bucket.ETag(withSid))
		objectsOnly := doc(bucket.Statement{Effect: bucket.Allow, Principal: only("alice"), Actions: bucket.NewStringSet("s3:GetObject"), Resources: bucket.NewStringSet(objectsARN)})
		require.NotEqual(t, bucket.ETag(d), bucket.ETag(objectsOnly))
		older := d
		older.Version = bucket.Version2008
		require.NotEqual(t, bucket.ETag(d), bucket.ETag(older))
	})

	t.Run("pins the canonical hashes", func(t *testing.T) {
		// Hilt compares stored tags against freshly computed ones for If-Match,
		// and Ingot answers clients with freshly computed ones, so these values
		// must never change without a deliberate decision. Both were computed
		// from hand-assembled DAG-JSON bytes:
		// {"statement":[],"version":""} and
		// {"statement":[<the statement above>],"version":"2012-10-17"}.
		require.Equal(t, `"baguqeerapfhqlt74verbk25awofci4fsi4dewjbihdbnexfwphaywy4srd2q"`, bucket.ETag(bucket.BucketPolicy{}))
		require.Equal(t, `"baguqeeramb7bdvrj6whijwstk5ktmw3z3wrwg64cf4xfj6qb2xxgsxsj7emq"`, bucket.ETag(d))
		require.Equal(t, bucket.ETag(bucket.BucketPolicy{}), bucket.ETag(bucket.BucketPolicy{Statements: []bucket.Statement{}}))
		require.Equal(t, `"`+dagJSONCID(t, `{"statement":[],"version":""}`).String()+`"`, bucket.ETag(bucket.BucketPolicy{}))
	})
}

// dagJSONCID is the CID the ETag is built from: CIDv1, dag-json, sha2-256
// over raw, which the tests assemble by hand so that the pins do not depend
// on the generated encoder.
func dagJSONCID(t *testing.T, raw string) cid.Cid {
	t.Helper()
	sum, err := mh.Sum([]byte(raw), mh.SHA2_256, -1)
	require.NoError(t, err)
	return cid.NewCidV1(cid.DagJSON, sum)
}

// dagJSONSet is the DAG-JSON encoding of a set: an object whose keys are the
// members, sorted, each mapped to {}.
func dagJSONSet(members ...string) string {
	members = slices.Clone(members)
	slices.Sort(members)
	for i, m := range members {
		members[i] = `"` + m + `":{}`
	}
	return "{" + strings.Join(members, ",") + "}"
}

// dagJSONStatement is the DAG-JSON encoding of a statement: keys in sorted
// order, sid omitted when empty, principals, actions and resources as sets,
// the wildcard principal as the set {"*"}.
func dagJSONStatement(sid string, effect bucket.Effect, principals, actions, resources []string) string {
	raw := `{"action":` + dagJSONSet(actions...) + `,"effect":"` + string(effect) + `","principal":` + dagJSONSet(principals...) + `,"resource":` + dagJSONSet(resources...)
	if sid != "" {
		raw += `,"sid":"` + sid + `"`
	}
	return raw + "}"
}

// dagJSONPolicy is the DAG-JSON encoding of a policy: its statements as a
// list in the order of their CIDs, then its version.
func dagJSONPolicy(t *testing.T, version string, statements ...string) string {
	t.Helper()
	statements = slices.Clone(statements)
	slices.SortFunc(statements, func(a, b string) int { return strings.Compare(dagJSONCID(t, a).String(), dagJSONCID(t, b).String()) })
	return `{"statement":[` + strings.Join(statements, ",") + `],"version":"` + version + `"}`
}
