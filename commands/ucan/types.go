package ucan

import "github.com/ipfs/go-cid"

type ConcludeArguments struct {
	Receipt cid.Cid `cborgen:"receipt" dagjsongen:"receipt"`
}
