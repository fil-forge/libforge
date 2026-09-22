// Package metrics defines the usage metering capabilities. The space the
// samples describe is the invocation subject, so it never appears in the
// arguments.
package metrics

// SampleArguments is the argument shape of `/metrics/sample`. It asks for a
// usage time series for the invocation subject, covering [From, To) in buckets
// of Window seconds.
//
// All three fields are required and are expressed in seconds: From and To are
// Unix timestamps, From inclusive and To exclusive, and Window is a duration.
// Window need not divide To-From evenly — the final bucket is then short and
// ends at To.
//
// The series the service returns is dense: a bucket in which nothing happened
// still yields a sample. Consumers take an unweighted mean over the stored-bytes
// series, so a bucket dropped for want of activity would reweight that mean.
type SampleArguments struct {
	// From is the start of the range, inclusive, as a Unix timestamp in seconds.
	From int64 `cborgen:"from" dagjsongen:"from"`
	// To is the end of the range, exclusive, as a Unix timestamp in seconds.
	To int64 `cborgen:"to" dagjsongen:"to"`
	// Window is the width of one bucket, in seconds. It must be positive.
	Window int64 `cborgen:"window" dagjsongen:"window"`
}

// SampleOK is the success return for `/metrics/sample`. Samples holds exactly
// one entry per bucket in [From, To), ordered by ascending Timestamp, with no
// gaps.
//
// From, To and Window restate the range the samples cover. From and Window
// always equal the request's. To is the requested To clamped to the service's
// current time: a bucket that has not closed has no value to report, and
// padding one with the latest reading would bias a consumer's average. A
// request whose range lies entirely in the future returns no samples.
type SampleOK struct {
	// From is the start of the range covered, inclusive, as a Unix timestamp in
	// seconds.
	From int64 `cborgen:"from" dagjsongen:"from"`
	// To is the end of the range covered, exclusive, as a Unix timestamp in
	// seconds. It is the requested To clamped to the service's current time.
	To int64 `cborgen:"to" dagjsongen:"to"`
	// Window is the width of one bucket, in seconds.
	Window int64 `cborgen:"window" dagjsongen:"window"`
	// Samples is one entry per bucket, ascending by Timestamp.
	Samples []SampleItem `cborgen:"samples" dagjsongen:"samples"`
}

// SampleItem is one bucket of the series.
//
// Timestamp is the end of the bucket rather than its start. Consumers key a
// sample to the instant its window closes, and a start-of-window timestamp
// lands the value in the neighbouring bucket when several series are collapsed
// onto one grid.
//
// BytesStored is a gauge: the bytes the space holds as of Timestamp, counting
// everything stored and not yet removed before that instant. BytesIngested is a
// flow: the bytes added during the bucket. Removals do not reduce BytesIngested;
// they show up in the next BytesStored.
//
// There is no egress here. Egress is accounted for by the egress tracking
// service, which sees retrieval receipts this service never handles.
type SampleItem struct {
	// Timestamp is the end of the bucket, as a Unix timestamp in seconds.
	Timestamp int64 `cborgen:"timestamp" dagjsongen:"timestamp"`
	// BytesStored is the bytes the space holds at Timestamp.
	BytesStored uint64 `cborgen:"bytesStored" dagjsongen:"bytesStored"`
	// BytesIngested is the bytes added to the space during the bucket.
	BytesIngested uint64 `cborgen:"bytesIngested" dagjsongen:"bytesIngested"`
}
