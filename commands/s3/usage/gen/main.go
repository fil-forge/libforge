//go:generate go run -tags codegen .

package main

import (
	"os"

	jsg "github.com/alanshaw/dag-json-gen"
	"github.com/fil-forge/libforge/commands/s3/usage"
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
	mapModels := []any{
		usage.ReportArguments{},
		usage.ReportOK{},
	}
	// The egress entries are tuple encoded: they repeat once for each space and
	// each hour, and a map would repeat its key names with them.
	tupleModels := []any{
		usage.SpaceEgress{},
		usage.HourEgress{},
	}
	const (
		cborTuples = "../cbor_gen.tuples.go"
		cborMaps   = "../cbor_gen.maps.go"
		jsonTuples = "../json_gen.tuples.go"
		jsonMaps   = "../json_gen.maps.go"
	)
	if err := cbg.WriteTupleEncodersToFile(cborTuples, "usage", tupleModels...); err != nil {
		panic(err)
	}
	if err := cbg.WriteMapEncodersToFile(cborMaps, "usage", mapModels...); err != nil {
		panic(err)
	}
	if err := jsg.WriteTupleEncodersToFile(jsonTuples, "usage", tupleModels...); err != nil {
		panic(err)
	}
	if err := jsg.WriteMapEncodersToFile(jsonMaps, "usage", mapModels...); err != nil {
		panic(err)
	}
	tag(cborTuples)
	tag(cborMaps)
	tag(jsonTuples)
	tag(jsonMaps)
}
