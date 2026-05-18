//go:build !codegen

package sign

import "github.com/fil-forge/libforge/capabilities"

const (
	DataSetCreateCommand        = "/pdp/sign/dataset/create"
	DataSetDeleteCommand        = "/pdp/sign/dataset/delete"
	PiecesAddCommand            = "/pdp/sign/pieces/add"
	PiecesRemoveScheduleCommand = "/pdp/sign/pieces/remove/schedule"
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
	DataSetCreate        = capabilities.MustNew[*DataSetCreateArguments](DataSetCreateCommand)
	DataSetDelete        = capabilities.MustNew[*DataSetDeleteArguments](DataSetDeleteCommand)
	PiecesAdd            = capabilities.MustNew[*PiecesAddArguments](PiecesAddCommand)
	PiecesRemoveSchedule = capabilities.MustNew[*PiecesRemoveScheduleArguments](PiecesRemoveScheduleCommand)
)
