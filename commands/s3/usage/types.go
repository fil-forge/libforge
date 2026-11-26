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
	// Objects is the object counts of the gateway's buckets, one entry for
	// each bucket's space. A space appears at most once.
	Objects []SpaceObjects `cborgen:"objects" dagjsongen:"objects"`
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
	// Bytes is the bytes sent in the hour so far. It is a total that only
	// grows, and the service keeps the largest reported for the hour: an hour
	// still in progress is reported again as it grows.
	Bytes uint64
}

// SpaceObjects is the object counts of one space, by the hour.
//
// It is tuple encoded, as [Space, Hours], so that a space's DID is sent once
// however many hours it has.
type SpaceObjects struct {
	// Space is the bucket's space.
	Space did.DID
	// Hours is the counts as of each hour reported. An hour appears at most
	// once.
	Hours []HourObjects
}

// HourObjects is the object counts of one space as of one hour.
//
// The two counts are totals over the life of the space, which only grow, so
// the service keeps the largest of each reported for the hour: an hour still
// in progress is reported again as they grow. The objects the space holds
// are Added less Removed, read from the latest hour reported. An hour in
// which nothing changed is not reported.
//
// It is tuple encoded, as [Hour, Added, Removed], because it repeats once
// for each hour of each space and a map would carry its key names along with
// it.
type HourObjects struct {
	// Hour is the start of the hour, as a Unix timestamp in seconds. It is a
	// whole number of hours.
	Hour int64
	// Added is the objects the space has ever gained, as of the end of the
	// hour or of the report if the hour is still in progress. An object is
	// whatever the gateway counts as one, such as a version of a key or a
	// delete marker.
	Added uint64
	// Removed is the objects the space has ever lost, as of the same
	// instant. It is not a count of deletes: a delete that leaves a delete
	// marker adds an object. A service reads a space that has lost more than
	// it gained as holding none.
	Removed uint64
}

// Names of the warnings a report can come back with.
const (
	// UnknownBucketWarningName says the service has no record of the space's
	// bucket, so its counts were not stored.
	UnknownBucketWarningName = "UnknownBucket"
	// BucketRegionMismatchWarningName says another gateway serves the space's
	// bucket, so its counts were not stored.
	BucketRegionMismatchWarningName = "BucketRegionMismatch"
)

// ReportOK is the success return for `/s3/usage/report`.
type ReportOK struct {
	// Warnings is what the service has to say about a report it accepted,
	// such as the spaces whose counts it did not store. The rest of the report
	// was stored. It is empty when there is nothing to say.
	//
	// An accepted report settles every count in it, warned about or not: the
	// gateway does not send them again. A warning is for the gateway to log
	// and count, by its name.
	Warnings []Warning `cborgen:"warnings" dagjsongen:"warnings"`
}

// Warning is one thing the service has to say about a space in a report.
type Warning struct {
	// Name is the machine readable name of the warning. A gateway may meet a
	// name it does not know, from a newer service.
	Name string `cborgen:"name" dagjsongen:"name"`
	// Message is a human readable description of the warning.
	Message string `cborgen:"message" dagjsongen:"message"`
	// Space is the space the warning is about.
	Space did.DID `cborgen:"space" dagjsongen:"space"`
}
