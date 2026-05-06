package access

import (
	adm "github.com/fil-forge/libforge/capabilities/access/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

type (
	RequestArguments = adm.RequestArgumentsModel
	RequestOK        = adm.RequestOKModel
)

// RequestFactKey is the key in metadata in any delegation created by a
// successful access request. The value is a link back to the `/access/request`
// invocation.
const RequestMetaKey = "accessRequest"

const RequestCommand = "/access/request"

// Request can be invoked by an agent to request set of capabilities from the
// account.
var Request, _ = bindcap.New[*RequestArguments](RequestCommand)
