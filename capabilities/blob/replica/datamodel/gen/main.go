//go:generate go run .

package main

import (
	rdm "github.com/fil-forge/libforge/capabilities/blob/replica/datamodel"
	cbg "github.com/whyrusleeping/cbor-gen"
)

func main() {
	if err := cbg.WriteMapEncodersToFile("../cbor_gen.go", "datamodel",
		rdm.AllocateArgumentsModel{},
		rdm.BlobModel{},
		rdm.AllocateOKModel{},
		rdm.TransferArgumentsModel{},
		rdm.TransferOKModel{},
	); err != nil {
		panic(err)
	}
}
