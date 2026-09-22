//go:generate go run -tags codegen .

package main

import (
	"os"

	jsg "github.com/alanshaw/dag-json-gen"
	"github.com/fil-forge/libforge/commands/metrics"
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
	models := []any{
		metrics.SampleArguments{},
		metrics.SampleOK{},
		metrics.SampleItem{},
	}
	const (
		cborFile = "../cbor_gen.go"
		jsonFile = "../json_gen.go"
	)
	if err := cbg.WriteMapEncodersToFile(cborFile, "metrics", models...); err != nil {
		panic(err)
	}
	if err := jsg.WriteMapEncodersToFile(jsonFile, "metrics", models...); err != nil {
		panic(err)
	}
	tag(cborFile)
	tag(jsonFile)
}
