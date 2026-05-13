//go:generate go run .

package main

import (
	jsg "github.com/alanshaw/dag-json-gen"
	"github.com/fil-forge/libforge/capabilities/pdp/datamodel"
	cbg "github.com/whyrusleeping/cbor-gen"
)

func main() {
	models := []any{
		datamodel.AcceptArgumentsModel{},
		datamodel.AcceptOKModel{},
		datamodel.InfoArgumentsModel{},
		datamodel.InfoAcceptedAggregateModel{},
		datamodel.InfoOKModel{},
	}

	if err := cbg.WriteMapEncodersToFile("../cbor_gen.go", "datamodel", models...); err != nil {
		panic(err)
	}

	if err := jsg.WriteMapEncodersToFile("../json_gen.go", "datamodel", models...); err != nil {
		panic(err)
	}
}
