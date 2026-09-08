//go:generate go run -tags codegen .

package main

import (
	"os"

	jsg "github.com/alanshaw/dag-json-gen"
	"github.com/fil-forge/libforge/commands/routing"
	cbg "github.com/whyrusleeping/cbor-gen"
)

const buildTag = "//go:build !codegen\n\n"

func tag(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, append([]byte(buildTag), data...), 0644); err != nil {
		panic(err)
	}
}

func main() {
	// CandidateSet has a hand-written codec (see codec.go) because cbor-gen /
	// dag-json-gen do not support DID-keyed maps.
	models := []any{
		routing.Candidate{},
		routing.PutArguments{},
		routing.UseArguments{},
	}
	const (
		cborFile = "../cbor_gen.go"
		jsonFile = "../json_gen.go"
	)
	if err := cbg.WriteMapEncodersToFile(cborFile, "routing", models...); err != nil {
		panic(err)
	}
	if err := jsg.WriteMapEncodersToFile(jsonFile, "routing", models...); err != nil {
		panic(err)
	}
	tag(cborFile)
	tag(jsonFile)
}
