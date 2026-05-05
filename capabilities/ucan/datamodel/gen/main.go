//go:generate go run .

package main

import (
	jsg "github.com/alanshaw/dag-json-gen"
	udm "github.com/fil-forge/libforge/capabilities/ucan/datamodel"
	cbg "github.com/whyrusleeping/cbor-gen"
)

func main() {
	models := []any{
		udm.ConcludeArgumentsModel{},
	}
	if err := cbg.WriteMapEncodersToFile("../cbor_gen.go", "datamodel", models...); err != nil {
		panic(err)
	}
	if err := jsg.WriteMapEncodersToFile("../json_gen.go", "datamodel", models...); err != nil {
		panic(err)
	}
}
