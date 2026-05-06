package access

import (
	adm "github.com/fil-forge/libforge/capabilities/access/datamodel"
	cdm "github.com/fil-forge/libforge/capabilities/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

type (
	ClaimArguments = cdm.UnitModel
	ClaimOK        = adm.ClaimOKModel
)

const ClaimCommand = "/access/claim"

// Claim can be invoked by an agent to claim a set of delegations from the
// account.
var Claim, _ = bindcap.New[*ClaimArguments](ClaimCommand)
