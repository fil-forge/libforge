package principal

import "github.com/fil-forge/ucantone/did"

// InvalidateArguments are the arguments to `/principal/invalidate`.
type InvalidateArguments struct {
	// Tenant is the DID of the tenant the principal belongs to.
	Tenant did.DID `cborgen:"tenant" dagjsongen:"tenant"`
	// Principal is the identifier of the principal whose cached proofs are
	// void, unique within the tenant.
	Principal string `cborgen:"principal" dagjsongen:"principal"`
}
