package ucan

import "github.com/ipfs/go-cid"

// ConcludeArguments delivers receipts to an audience awaiting them. It is
// always a list, however many are delivered: a delivering agent often holds a
// receipt per blob of a large upload, and a round trip per receipt costs more
// than the work each one triggers. An empty list delivers nothing.
type ConcludeArguments struct {
	Receipts []cid.Cid `cborgen:"receipts" dagjsongen:"receipts"`
}

type RevokeArguments struct {
	// Revoke is the CID of the UCAN delegation to revoke.
	Revoke cid.Cid `cborgen:"revoke" dagjsongen:"revoke"`
	// Path is the delegation path to the UCAN delegation to revoke. The path is a
	// list of CIDs that represent the delegation chain from the root UCAN to the
	// UCAN being revoked.
	Path []cid.Cid `cborgen:"path" dagjsongen:"path"`
}
