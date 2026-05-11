package datamodel

import (
	"github.com/fil-forge/ucantone/did"
	cid "github.com/ipfs/go-cid"
)

type ConfirmArgumentsModel struct {
	// Link to the `/access/request` this invocation is confirming.
	Cause        cid.Cid                  `cborgen:"cause" dagjsongen:"cause"`
	Issuer       did.DID                  `cborgen:"iss" dagjsongen:"iss"`
	Audience     did.DID                  `cborgen:"aud" dagjsongen:"aud"`
	Attenuations []CapabilityRequestModel `cborgen:"att" dagjsongen:"att"`
}

type ConfirmOKModel = ClaimOKModel
