package datamodel

import "github.com/fil-forge/ucantone/did"

type InfoOKModel struct {
	Providers []did.DID `cborgen:"providers" dagjsongen:"providers"`
}
