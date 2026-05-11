package access

import (
	adm "github.com/fil-forge/libforge/capabilities/access/datamodel"
	"github.com/fil-forge/ucantone/errors"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

type (
	ConfirmArguments = adm.ConfirmArgumentsModel
	ConfirmOK        = adm.ConfirmOKModel
)

// ConfirmMetaKey is the key in metadata in any delegation created by a
// successful access request. The value is a link back to the `/access/confirm`
// invocation.
const ConfirmMetaKey = "accessConfirm"

const ConfirmCommand = "/access/confirm"

// Confirm can be invoked by an agent to confirm an access request.
var Confirm, _ = bindcap.New[*ConfirmArguments](ConfirmCommand)

const (
	InvalidAccessConfirmSubjectErrorName = "InvalidAccessConfirmSubject"
	InvalidAccessConfirmIssuerErrorName  = "InvalidAccessConfirmIssuer"
)

var (
	ErrInvalidAccessConfirmSubject = errors.New(InvalidAccessConfirmSubjectErrorName, "the subject of an access confirm invocation must be the service itself")
	ErrInvalidAccessConfirmIssuer  = errors.New(InvalidAccessConfirmIssuerErrorName, "the issuer of an access confirm invocation must be a valid mailto DID")
)
