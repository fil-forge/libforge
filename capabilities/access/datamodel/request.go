package datamodel

import (
	"github.com/fil-forge/ucantone/did"
	"github.com/fil-forge/ucantone/ucan"
	"github.com/ipfs/go-cid"
)

type RequestArgumentsModel struct {
	// DID of the Account authorization is requested from.
	Issuer did.DID `cborgen:"iss" dagjsongen:"iss"`
	// Capabilities agent wishes to be granted.
	Attenuations []CapabilityRequestModel `cborgen:"att" dagjsongen:"att"`
}

type CapabilityRequestModel struct {
	Command ucan.Command `cborgen:"cmd" dagjsongen:"cmd"`
}

type RequestOKModel struct {
	// Request is a link to the access request invocation.
	Request cid.Cid `cborgen:"req" dagjsongen:"req"`
	// Expiration is the time at which the confirmation will expire.
	Expiration int64 `cborgen:"exp" dagjsongen:"exp"`
}
