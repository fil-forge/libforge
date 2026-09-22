//go:build !codegen

package metrics_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/fil-forge/libforge/commands/metrics"
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

func TestSampleArgumentsRoundTrip(t *testing.T) {
	in := &metrics.SampleArguments{From: 1700000000, To: 1700003600, Window: 3600}
	require.Equal(t, `{"from":1700000000,"to":1700003600,"window":3600}`, roundTrip(t, in))
}

// provider is a fixed DID so the expected DAG-JSON below is stable.
var provider = did.MustParse("did:web:provider.example")

func TestSampleOKRoundTrip(t *testing.T) {
	in := &metrics.SampleOK{
		From:   1700000000,
		To:     1700007200,
		Window: 3600,
		Samples: metrics.SampleSet{Entries: map[did.DID][]metrics.SampleItem{
			provider: {
				{Timestamp: 1700003600, BytesStored: 1024, BytesIngested: 1024},
				{Timestamp: 1700007200, BytesStored: 512, BytesIngested: 0},
			},
		}},
	}
	// SampleItem is tuple encoded: [timestamp, bytesStored, bytesIngested].
	require.Equal(t,
		`{"from":1700000000,"samples":{"did:web:provider.example":[`+
			`[1700003600,1024,1024],`+
			`[1700007200,512,0]`+
			`]},"to":1700007200,"window":3600}`,
		roundTrip(t, in))
}

// A space provisioned with several providers carries a series each, on one
// shared bucket grid. The keys sort, so the encoding is deterministic.
func TestSampleOKSeveralProvidersRoundTrip(t *testing.T) {
	a := did.MustParse("did:web:a.example")
	b := did.MustParse("did:web:b.example")
	in := &metrics.SampleOK{
		From:   1700000000,
		To:     1700003600,
		Window: 3600,
		Samples: metrics.SampleSet{Entries: map[did.DID][]metrics.SampleItem{
			b: {{Timestamp: 1700003600, BytesStored: 2, BytesIngested: 2}},
			a: {{Timestamp: 1700003600, BytesStored: 1, BytesIngested: 1}},
		}},
	}
	require.Equal(t,
		`{"from":1700000000,"samples":{`+
			`"did:web:a.example":[[1700003600,1,1]],`+
			`"did:web:b.example":[[1700003600,2,2]]`+
			`},"to":1700003600,"window":3600}`,
		roundTrip(t, in))
}

// A range entirely in the future carries no samples. An absent set and an
// empty one are the same thing on the wire, so these assert the encoding and
// the decoded content rather than round-trip identity.
func TestSampleOKEmptySetRoundTrip(t *testing.T) {
	in := &metrics.SampleOK{From: 1700000000, To: 1700000000, Window: 3600}

	var jb bytes.Buffer
	require.NoError(t, in.MarshalDagJSON(&jb))
	require.Equal(t, `{"from":1700000000,"samples":{},"to":1700000000,"window":3600}`, jb.String())

	var fromJSON metrics.SampleOK
	require.NoError(t, fromJSON.UnmarshalDagJSON(bytes.NewReader(jb.Bytes())))
	require.Empty(t, fromJSON.Samples.Entries)

	var cb bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&cb))
	var fromCBOR metrics.SampleOK
	require.NoError(t, fromCBOR.UnmarshalCBOR(bytes.NewReader(cb.Bytes())))
	require.Empty(t, fromCBOR.Samples.Entries)
}

// A provider present with nothing to report keeps its key and an empty series.
func TestSampleOKEmptySeriesRoundTrip(t *testing.T) {
	in := &metrics.SampleOK{
		From: 1700000000, To: 1700000000, Window: 3600,
		Samples: metrics.SampleSet{Entries: map[did.DID][]metrics.SampleItem{provider: {}}},
	}

	var jb bytes.Buffer
	require.NoError(t, in.MarshalDagJSON(&jb))
	require.Equal(t,
		`{"from":1700000000,"samples":{"did:web:provider.example":[]},"to":1700000000,"window":3600}`,
		jb.String())

	var fromJSON metrics.SampleOK
	require.NoError(t, fromJSON.UnmarshalDagJSON(bytes.NewReader(jb.Bytes())))
	require.Len(t, fromJSON.Samples.Entries, 1)
	require.Empty(t, fromJSON.Samples.Entries[provider])

	var cb bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&cb))
	var fromCBOR metrics.SampleOK
	require.NoError(t, fromCBOR.UnmarshalCBOR(bytes.NewReader(cb.Bytes())))
	require.Len(t, fromCBOR.Samples.Entries, 1)
	require.Empty(t, fromCBOR.Samples.Entries[provider])
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
	samples := make([]metrics.SampleItem, 0, count)
	for i := range count {
		samples = append(samples, metrics.SampleItem{
			Timestamp:     in.From + int64(i+1)*hour,
			BytesStored:   uint64(i) * 1 << 20,
			BytesIngested: 1 << 20,
		})
	}
	in.Samples = metrics.SampleSet{Entries: map[did.DID][]metrics.SampleItem{provider: samples}}
	in.To = in.From + count*hour

	var cb bytes.Buffer
	require.NoError(t, in.MarshalCBOR(&cb))
	var out metrics.SampleOK
	require.NoError(t, out.UnmarshalCBOR(bytes.NewReader(cb.Bytes())))
	require.Len(t, out.Samples.Entries[provider], count)
	require.Equal(t, *in, out)
	t.Logf("%d samples encode to %d bytes of CBOR", count, cb.Len())
}
