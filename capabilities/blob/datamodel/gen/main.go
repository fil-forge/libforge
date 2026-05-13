//go:generate go run .

package main

import (
	jsg "github.com/alanshaw/dag-json-gen"
	bdm "github.com/fil-forge/libforge/capabilities/blob/datamodel"
	cbg "github.com/whyrusleeping/cbor-gen"
)

func main() {
	models := []any{
		bdm.AllocateArgumentsModel{},
		bdm.BlobModel{},
		bdm.AllocateOKModel{},
		bdm.BlobAddressModel{},
		bdm.AcceptArgumentsModel{},
		bdm.AcceptOKModel{},
		bdm.AddArgumentsModel{},
		bdm.AddOKModel{},
		bdm.RemoveArgumentsModel{},
		bdm.ReplicateArgumentsModel{},
		bdm.ReplicateOKModel{},
		bdm.ListArgumentsModel{},
		bdm.ListOKModel{},
		bdm.ListBlobItem{},
		bdm.RetrieveArgumentsModel{},
		bdm.RetrieveBlobModel{},
		bdm.RetrieveOKModel{},
	}
	if err := cbg.WriteMapEncodersToFile("../cbor_gen.go", "datamodel", models...); err != nil {
		panic(err)
	}
	if err := jsg.WriteMapEncodersToFile("../json_gen.go", "datamodel", models...); err != nil {
		panic(err)
	}
}
