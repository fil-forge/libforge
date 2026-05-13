// Package sign exposes the four /pdp/sign/* UCAN capabilities used by piri
// to drive an eip712 signing service:
//
//   - /pdp/sign/dataset/create
//   - /pdp/sign/dataset/delete
//   - /pdp/sign/pieces/add
//   - /pdp/sign/pieces/remove/schedule
//
// Each returns an [AuthSignature] (an eip712-signed bytes payload).
package sign

import (
	sdm "github.com/fil-forge/libforge/capabilities/pdp/sign/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

const (
	DataSetCreateCommand        = "/pdp/sign/dataset/create"
	DataSetDeleteCommand        = "/pdp/sign/dataset/delete"
	PiecesAddCommand            = "/pdp/sign/pieces/add"
	PiecesRemoveScheduleCommand = "/pdp/sign/pieces/remove/schedule"
)

type (
	// DataSetCreateArguments / DataSetCreateOK are the args / response for
	// /pdp/sign/dataset/create.
	DataSetCreateArguments = sdm.DataSetCreateArgumentsModel
	DataSetCreateOK        = sdm.AuthSignatureModel

	// DataSetDeleteArguments / DataSetDeleteOK are the args / response for
	// /pdp/sign/dataset/delete.
	DataSetDeleteArguments = sdm.DataSetDeleteArgumentsModel
	DataSetDeleteOK        = sdm.AuthSignatureModel

	// PiecesAddArguments / PiecesAddOK are the args / response for
	// /pdp/sign/pieces/add.
	PiecesAddArguments = sdm.PiecesAddArgumentsModel
	PiecesAddOK        = sdm.AuthSignatureModel

	// PiecesRemoveScheduleArguments / PiecesRemoveScheduleOK are the args /
	// response for /pdp/sign/pieces/remove/schedule.
	PiecesRemoveScheduleArguments = sdm.PiecesRemoveScheduleArgumentsModel
	PiecesRemoveScheduleOK        = sdm.AuthSignatureModel

	// AuthSignature is the shared response shape.
	AuthSignature = sdm.AuthSignatureModel

	// Metadata is the eip712 metadata bag attached to every dataset and
	// piece signing request.
	Metadata = sdm.MetadataModel

	// PieceProofs wraps the list of `blob/accept` invocation CIDs proving
	// the sub-pieces of one piece in a /pdp/sign/pieces/add request.
	PieceProofs = sdm.PieceProofsModel
)

var (
	DataSetCreate, _        = bindcap.New[*DataSetCreateArguments](DataSetCreateCommand)
	DataSetDelete, _        = bindcap.New[*DataSetDeleteArguments](DataSetDeleteCommand)
	PiecesAdd, _            = bindcap.New[*PiecesAddArguments](PiecesAddCommand)
	PiecesRemoveSchedule, _ = bindcap.New[*PiecesRemoveScheduleArguments](PiecesRemoveScheduleCommand)
)
