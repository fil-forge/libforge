package datamodel

import (
	"github.com/fil-forge/ucantone/did"
)

type AddArgumentsModel struct {
	Provider did.DID `cborgen:"provider" dagjsongen:"provider"`
	Consumer did.DID `cborgen:"consumer" dagjsongen:"consumer"`
}

type AddOKModel struct {
	ID string `cborgen:"id" dagjsongen:"id"`
}
