//go:build !codegen

package metrics_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/fil-forge/libforge/commands/metrics"
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

func TestSampleArgumentsRoundTrip(t *testing.T) {
	in := &metrics.SampleArguments{From: 1700000000, To: 1700003600, Window: 3600}
	require.Equal(t, `{"from":1700000000,"to":1700003600,"window":3600}`, roundTrip(t, in))
}

func TestSampleOKRoundTrip(t *testing.T) {
	in := &metrics.SampleOK{
		From:   1700000000,
		To:     1700007200,
		Window: 3600,
		Samples: []metrics.SampleItem{
			{Timestamp: 1700003600, BytesStored: 1024, BytesIngested: 1024},
			{Timestamp: 1700007200, BytesStored: 512, BytesIngested: 0},
		},
	}
	require.Equal(t,
		`{"from":1700000000,"samples":[`+
			`{"bytesIngested":1024,"bytesStored":1024,"timestamp":1700003600},`+
			`{"bytesIngested":0,"bytesStored":512,"timestamp":1700007200}`+
			`],"to":1700007200,"window":3600}`,
		roundTrip(t, in))
}

// A range entirely in the future carries no samples, so the empty series has to
// survive both codecs.
func TestSampleOKEmptySeriesRoundTrip(t *testing.T) {
	in := &metrics.SampleOK{From: 1700000000, To: 1700000000, Window: 3600}
	require.Equal(t, `{"from":1700000000,"samples":[],"to":1700000000,"window":3600}`, roundTrip(t, in))

	var cb bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&cb))
	var out metrics.SampleOK
	require.NoError(t, out.UnmarshalCBOR(bytes.NewReader(cb.Bytes())))
	require.Empty(t, out.Samples)
}

// 768 samples is a 32 day range at hourly granularity, the largest series
// consumers ask for. Both codecs cap arrays at 8192 elements, so this stays
// well inside the limit — but the encoded size is worth knowing.
func TestSampleOKLargeSeriesRoundTrip(t *testing.T) {
	const (
		hour  = int64(3600)
		count = 768
	)
	in := &metrics.SampleOK{From: 1700000000, Window: hour}
	for i := range count {
		in.Samples = append(in.Samples, metrics.SampleItem{
			Timestamp:     in.From + int64(i+1)*hour,
			BytesStored:   uint64(i) * 1 << 20,
			BytesIngested: 1 << 20,
		})
	}
	in.To = in.From + count*hour

	var cb bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&cb))
	var out metrics.SampleOK
	require.NoError(t, out.UnmarshalCBOR(bytes.NewReader(cb.Bytes())))
	require.Len(t, out.Samples, count)
	require.Equal(t, *in, out)
	t.Logf("%d samples encode to %d bytes of CBOR", count, cb.Len())
}
