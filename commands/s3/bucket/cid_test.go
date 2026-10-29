package bucket_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fil-forge/libforge/commands/s3/bucket"
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
