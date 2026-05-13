//go:generate go run .

package main

import (
	jsg "github.com/alanshaw/dag-json-gen"
	"github.com/fil-forge/libforge/capabilities/pdp/sign/datamodel"
	cbg "github.com/whyrusleeping/cbor-gen"
)

func main() {
	models := []any{
		datamodel.AuthSignatureModel{},
		datamodel.MetadataModel{},
		datamodel.PieceProofsModel{},
		datamodel.DataSetCreateArgumentsModel{},
		datamodel.DataSetDeleteArgumentsModel{},
		datamodel.PiecesAddArgumentsModel{},
		datamodel.PiecesRemoveScheduleArgumentsModel{},
	}

	if err := cbg.WriteMapEncodersToFile("../cbor_gen.go", "datamodel", models...); err != nil {
		panic(err)
	}

	if err := jsg.WriteMapEncodersToFile("../json_gen.go", "datamodel", models...); err != nil {
		panic(err)
	}
}
