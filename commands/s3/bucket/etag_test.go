package bucket_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
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

func doc(statements ...bucket.Statement) bucket.PolicyDocument {
	return bucket.PolicyDocument{Version: bucket.Version2012, Statements: statements}
}

func TestStatementSet(t *testing.T) {
	a := doc(allow(only("alice"), "s3:GetObject"), deny(everyone, "s3:DeleteObject"))
	b := doc(deny(everyone, "s3:DeleteObject"), allow(only("alice"), "s3:GetObject"), deny(everyone, "s3:DeleteObject"))

	t.Run("every encoding orders statements by CID and drops duplicates", func(t *testing.T) {
		ja, err := json.Marshal(a)
		require.NoError(t, err)
		jb, err := json.Marshal(b)
		require.NoError(t, err)
		require.Equal(t, string(ja), string(jb))
		require.Equal(t, 2, strings.Count(string(jb), `"Effect"`))

		var da, db, ca, cb bytes.Buffer
		require.NoError(t, a.MarshalDagJSON(&da))
		require.NoError(t, b.MarshalDagJSON(&db))
		require.Equal(t, da.String(), db.String())
		require.NoError(t, a.MarshalCBOR(&ca))
		require.NoError(t, b.MarshalCBOR(&cb))
		require.Equal(t, ca.Bytes(), cb.Bytes())

		// The in-memory order is the caller's.
		require.Equal(t, bucket.Deny, b.Statements[0].Effect)
		require.Len(t, b.Statements, 3)
	})

	t.Run("decoding keeps the wire order, which is canonical", func(t *testing.T) {
		var cb bytes.Buffer
		require.NoError(t, b.MarshalCBOR(&cb))
		var back bucket.PolicyDocument
		require.NoError(t, back.UnmarshalCBOR(bytes.NewReader(cb.Bytes())))
		require.Len(t, back.Statements, 2)
		require.True(t, bucket.StatementCID(back.Statements[0]).String() < bucket.StatementCID(back.Statements[1]).String())
		require.Equal(t, a.Version, back.Version)

		var jb bytes.Buffer
		require.NoError(t, b.MarshalDagJSON(&jb))
		var backJSON bucket.PolicyDocument
		require.NoError(t, backJSON.UnmarshalDagJSON(bytes.NewReader(jb.Bytes())))
		require.Equal(t, back, backJSON)
	})

	t.Run("nil and empty encode the same", func(t *testing.T) {
		var n1, n2 bytes.Buffer
		require.NoError(t, (&bucket.PolicyDocument{}).MarshalDagJSON(&n1))
		require.NoError(t, (&bucket.PolicyDocument{Statements: bucket.StatementSet{}}).MarshalDagJSON(&n2))
		require.Equal(t, `{"statement":[],"version":""}`, n1.String())
		require.Equal(t, n1.String(), n2.String())
		raw, err := json.Marshal(bucket.PolicyDocument{})
		require.NoError(t, err)
		require.Equal(t, `{"Version":"","Statement":[]}`, string(raw))
	})

	t.Run("CBOR and DAG-JSON match the generated layout", func(t *testing.T) {
		// Captured from the generated encoders before the codec was hand-written:
		// CBOR keys shortest first (version, statement), DAG-JSON keys sorted.
		const cborHex = "a26776657273696f6e6a323031322d31302d31376973746174656d656e7481a466616374696f6ea16c73333a4765744f626a656374a06665666665637465416c6c6f77687265736f75726365a27361726e3a6177733a73333a3a3a70686f746f73a07561726e3a6177733a73333a3a3a70686f746f732f2aa0697072696e636970616ca165616c696365a0"
		const okCBORHex = "a166706f6c696379" + cborHex
		d := doc(allow(only("alice"), "s3:GetObject"))
		var cb bytes.Buffer
		require.NoError(t, d.MarshalCBOR(&cb))
		require.Equal(t, cborHex, hex.EncodeToString(cb.Bytes()))
		var back bucket.PolicyDocument
		require.NoError(t, back.UnmarshalCBOR(bytes.NewReader(cb.Bytes())))
		require.Equal(t, d, back)

		ok := bucket.PolicyOK{Policy: &d}
		cb.Reset()
		require.NoError(t, ok.MarshalCBOR(&cb))
		require.Equal(t, okCBORHex, hex.EncodeToString(cb.Bytes()))

		var jb bytes.Buffer
		require.NoError(t, d.MarshalDagJSON(&jb))
		require.Equal(t, `{"statement":[{"action":{"s3:GetObject":{}},"effect":"Allow","principal":{"alice":{}},"resource":{"arn:aws:s3:::photos":{},"arn:aws:s3:::photos/*":{}}}],"version":"2012-10-17"}`, jb.String())
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
		require.Equal(t, `"baguqeerapfhqlt74verbk25awofci4fsi4dewjbihdbnexfwphaywy4srd2q"`, bucket.ETag(bucket.PolicyDocument{}))
		require.Equal(t, `"baguqeeramb7bdvrj6whijwstk5ktmw3z3wrwg64cf4xfj6qb2xxgsxsj7emq"`, bucket.ETag(d))
		require.Equal(t, bucket.ETag(bucket.PolicyDocument{}), bucket.ETag(bucket.PolicyDocument{Statements: []bucket.Statement{}}))
		require.Equal(t, `"`+dagJSONCID(t, `{"statement":[],"version":""}`).String()+`"`, bucket.ETag(bucket.PolicyDocument{}))
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
