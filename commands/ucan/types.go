package ucan

import "github.com/ipfs/go-cid"

type ConcludeArguments struct {
	Receipt cid.Cid `cborgen:"receipt" dagjsongen:"receipt"`
}

type RevokeArguments struct {
	Revoke cid.Cid   `cborgen:"revoke" dagjsongen:"revoke"`
	Path   []cid.Cid `cborgen:"path" dagjsongen:"path"`
}
