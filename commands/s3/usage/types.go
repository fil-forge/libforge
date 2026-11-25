// Package usage defines the command an S3 gateway reports its usage counts
// with: the figures only the gateway knows, such as the bytes it sent to its
// clients.
package usage

import "github.com/fil-forge/ucantone/did"

const (
	// MaxSpaces is the most spaces one report can carry.
	MaxSpaces = 8192
	// MaxHours is the most hours one report can carry for a space.
	MaxHours = 8192
)

// ReportArguments is the argument shape of `/s3/usage/report`.
type ReportArguments struct {
	// Egress is the bytes of object data the gateway sent to its clients, one
	// entry for each bucket's space. A space appears at most once.
	Egress []SpaceEgress `cborgen:"egress" dagjsongen:"egress"`
}

// SpaceEgress is the egress of one space, by the hour.
//
// It is tuple encoded, as [Space, Hours], so that a space's DID is sent once
// however many hours it has.
type SpaceEgress struct {
	// Space is the bucket's space.
	Space did.DID
	// Hours is the egress of each hour reported. An hour appears at most once.
	Hours []HourEgress
}

// HourEgress is the egress of one hour.
//
// It is tuple encoded, as [Hour, Bytes], because it repeats once for each hour
// of each space and a map would carry its key names along with it.
type HourEgress struct {
	// Hour is the start of the hour, as a Unix timestamp in seconds. It is a
	// whole number of hours.
	Hour int64
	// Bytes is the bytes sent in the hour so far. It is a total, and replaces
	// any earlier report for the same hour: an hour still in progress is
	// reported again as it grows.
	Bytes uint64
}

// ReportOK is the success return for `/s3/usage/report`.
type ReportOK struct {
	// Skipped is the spaces whose counts were not stored, because the service
	// does not know the space or another gateway serves its bucket. The other
	// counts in the report were stored. The gateway does not send a skipped
	// count again.
	Skipped []did.DID `cborgen:"skipped" dagjsongen:"skipped"`
}
