//go:generate go run .

package main

import (
	jsg "github.com/alanshaw/dag-json-gen"
	"github.com/fil-forge/libforge/merkletree"
	cbg "github.com/whyrusleeping/cbor-gen"
)

func main() {
	models := []any{
		merkletree.ProofData{},
	}

	if err := cbg.WriteMapEncodersToFile("../cbor_gen.go", "merkletree", models...); err != nil {
		panic(err)
	}

	if err := jsg.WriteMapEncodersToFile("../json_gen.go", "merkletree", models...); err != nil {
		panic(err)
	}
}
