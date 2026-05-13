package blob

import (
	bdm "github.com/fil-forge/libforge/capabilities/blob/datamodel"
	"github.com/fil-forge/ucantone/validator/bindcap"
)

const RetrieveCommand = "/blob/retrieve"

type (
	RetrieveArguments = bdm.RetrieveArgumentsModel
	RetrieveBlob      = bdm.RetrieveBlobModel
	RetrieveOK        = bdm.RetrieveOKModel
)

// Retrieve is the service-level retrieval capability (e.g. used by the
// indexer to fetch content claims from a Piri node). It is NOT space-scoped:
// any holder of a valid delegation for `/blob/retrieve` may fetch the blob
// by digest, regardless of which space it was originally stored under.
//
// For user-facing retrieval that requires an allocation in a specific space
// see `libforge/capabilities/content.Retrieve` (the `/content/retrieve`
// capability).
var Retrieve, _ = bindcap.New[*RetrieveArguments](RetrieveCommand)
