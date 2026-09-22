//go:build !codegen

package metrics

import (
	"github.com/fil-forge/ucantone/binding"
	"github.com/fil-forge/ucantone/ucan/command"
)

// Sample returns a usage time series for the space that is the invocation
// subject. The service buckets the series itself, so the caller receives one
// sample per window with no gaps.
var Sample = binding.Bind[*SampleArguments, *SampleOK](command.MustParse("/metrics/sample"))

const (
	// InvalidRangeErrorName is returned when the requested range is not a
	// non-empty interval: To must be after From, and both must be positive Unix
	// timestamps.
	InvalidRangeErrorName = "InvalidRange"
	// InvalidWindowErrorName is returned when the requested window is not a
	// positive, representable number of seconds.
	InvalidWindowErrorName = "InvalidWindow"
	// TooManySamplesErrorName is returned when the range divided by the window
	// exceeds the number of samples the service returns in one response. The
	// message carries the limit. Callers should widen the window, or split the
	// range across several requests.
	TooManySamplesErrorName = "TooManySamples"
	// UsageUnstableErrorName is returned when the service could not read the
	// space's usage consistently, because the space was written to throughout
	// the attempt. It is retryable.
	UsageUnstableErrorName = "UsageUnstable"
	// RangeTooBusyErrorName is returned when the range holds more recorded
	// changes than the service will scan in one response. Callers should split
	// the range across several requests.
	RangeTooBusyErrorName = "RangeTooBusy"
)
