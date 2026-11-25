//go:build !codegen

package usage_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/fil-forge/libforge/commands/s3/usage"
	"github.com/fil-forge/ucantone/did"
	"github.com/stretchr/testify/require"
)

type wire interface {
	MarshalCBOR(w io.Writer) error
	UnmarshalCBOR(r io.Reader) error
	MarshalDagJSON(w io.Writer) error
	UnmarshalDagJSON(r io.Reader) error
}

// roundTrip encodes in as CBOR and DAG-JSON, decodes each and asserts equality.
// It returns the DAG-JSON encoding.
func roundTrip[T any, PT interface {
	*T
	wire
}](t *testing.T, in PT) string {
	t.Helper()
	var cb bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&cb))
	outCBOR := PT(new(T))
	require.NoError(t, outCBOR.UnmarshalCBOR(bytes.NewReader(cb.Bytes())))
	require.True(t, reflect.DeepEqual(*in, *outCBOR), "CBOR round-trip mismatch:\n got %#v\nwant %#v", *outCBOR, *in)

	var jb bytes.Buffer
	require.NoError(t, in.MarshalDagJSON(&jb))
	outJSON := PT(new(T))
	require.NoError(t, outJSON.UnmarshalDagJSON(bytes.NewReader(jb.Bytes())), "json: %s", jb.String())
	require.True(t, reflect.DeepEqual(*in, *outJSON), "DAG-JSON round-trip mismatch:\n got %#v\nwant %#v", *outJSON, *in)
	return jb.String()
}

const (
	spaceA = "did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK"
	spaceB = "did:key:z6MkpTHR8VNsBxYAAWHut2Geadd9jSwuBV8xRoAnwWsdvktH"
)

func TestReportArgumentsRoundTrip(t *testing.T) {
	a, err := did.Parse(spaceA)
	require.NoError(t, err)
	b, err := did.Parse(spaceB)
	require.NoError(t, err)
	in := &usage.ReportArguments{
		Egress: []usage.SpaceEgress{
			{Space: a, Hours: []usage.HourEgress{
				{Hour: 1795600800, Bytes: 5368709120},
				{Hour: 1795604400, Bytes: 1048576},
			}},
			{Space: b, Hours: []usage.HourEgress{
				{Hour: 1795600800, Bytes: 4096},
			}},
		},
	}
	// The entries are tuples: [space, [[hour, bytes], ...]].
	require.Equal(t,
		`{"egress":[`+
			`["`+spaceA+`",[[1795600800,5368709120],[1795604400,1048576]]],`+
			`["`+spaceB+`",[[1795600800,4096]]]`+
			`]}`,
		roundTrip(t, in))
}

func TestReportOKRoundTrip(t *testing.T) {
	b, err := did.Parse(spaceB)
	require.NoError(t, err)
	in := &usage.ReportOK{Warnings: []usage.Warning{{
		Name:    usage.UnknownBucketWarningName,
		Message: "no bucket has this space",
		Space:   b,
	}}}
	require.Equal(t,
		`{"warnings":[{"message":"no bucket has this space","name":"UnknownBucket","space":"`+spaceB+`"}]}`,
		roundTrip(t, in))
}

// A report with nothing to warn about says so with an empty list, and a
// reader gets no warnings back from it.
func TestReportOKWithNoWarnings(t *testing.T) {
	for _, in := range []*usage.ReportOK{{}, {Warnings: []usage.Warning{}}} {
		var cb bytes.Buffer
		require.NoError(t, in.MarshalCBOR(&cb))
		var out usage.ReportOK
		require.NoError(t, out.UnmarshalCBOR(&cb))
		require.Empty(t, out.Warnings)

		var jb bytes.Buffer
		require.NoError(t, in.MarshalDagJSON(&jb))
		require.Equal(t, `{"warnings":[]}`, jb.String())
	}
}

// A report over the limits is refused when it is written, so a gateway
// learns to split it before anything is sent.
func TestReportArgumentsRefusesTooManyEntries(t *testing.T) {
	a, err := did.Parse(spaceA)
	require.NoError(t, err)

	hours := make([]usage.HourEgress, usage.MaxHours+1)
	in := &usage.ReportArguments{Egress: []usage.SpaceEgress{{Space: a, Hours: hours}}}
	require.Error(t, in.MarshalCBOR(io.Discard))

	in = &usage.ReportArguments{Egress: make([]usage.SpaceEgress, usage.MaxSpaces+1)}
	require.Error(t, in.MarshalCBOR(io.Discard))
}
