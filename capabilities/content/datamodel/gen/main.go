//go:generate go run .

package main

import (
	jsg "github.com/alanshaw/dag-json-gen"
	dm "github.com/fil-forge/libforge/capabilities/content/datamodel"
	cbg "github.com/whyrusleeping/cbor-gen"
)

func main() {
	mapModels := []any{
		dm.BlobModel{},
		dm.RetrieveArgumentsModel{},
	}
	tupleModels := []any{
		dm.RangeModel{},
	}

	if err := cbg.WriteTupleEncodersToFile("../cbor_gen.tuples.go", "datamodel", tupleModels...); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile("../cbor_gen.maps.go", "datamodel", mapModels...); err != nil {
		panic(err)
	}

	if err := jsg.WriteTupleEncodersToFile("../json_gen.tuples.go", "datamodel", tupleModels...); err != nil {
		panic(err)
	}

	if err := jsg.WriteMapEncodersToFile("../json_gen.maps.go", "datamodel", mapModels...); err != nil {
		panic(err)
	}
}
