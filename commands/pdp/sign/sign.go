//go:build !codegen

package sign

import (
	"github.com/fil-forge/libforge/commands"
)

// Every /pdp/sign/* operation returns the same shape; these per-operation
// labels exist so call sites can keep the operation name in the type.
type (
	DataSetCreateOK        = AuthSignature
	DataSetDeleteOK        = AuthSignature
	PiecesAddOK            = AuthSignature
	PiecesRemoveScheduleOK = AuthSignature
)

var (
	DataSetCreate        = commands.MustParse[*DataSetCreateArguments, *DataSetCreateOK]("/pdp/sign/dataset/create")
	DataSetDelete        = commands.MustParse[*DataSetDeleteArguments, *DataSetDeleteOK]("/pdp/sign/dataset/delete")
	PiecesAdd            = commands.MustParse[*PiecesAddArguments, *PiecesAddOK]("/pdp/sign/pieces/add")
	PiecesRemoveSchedule = commands.MustParse[*PiecesRemoveScheduleArguments, *PiecesRemoveScheduleOK]("/pdp/sign/pieces/remove/schedule")
)
