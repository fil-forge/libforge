//go:generate go run .

package main

import (
	jsg "github.com/alanshaw/dag-json-gen"
	adm "github.com/fil-forge/libforge/capabilities/ucan/attest/datamodel"
	cbg "github.com/whyrusleeping/cbor-gen"
)

func main() {
	models := []any{
		adm.ProofArgumentsModel{},
	}
	if err := cbg.WriteMapEncodersToFile("../cbor_gen.go", "datamodel", models...); err != nil {
		panic(err)
	}
	if err := jsg.WriteMapEncodersToFile("../json_gen.go", "datamodel", models...); err != nil {
		panic(err)
	}
}
