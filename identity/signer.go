package identity

import (
	"github.com/fil-forge/ucantone/multikey"
)

// Signer is a [multikey.Signer] that is safe to pass to formatting and logging
// functions. Multikey signers are byte slices holding the private key, so
// formatting one directly writes the key material into the output.
//
// Prefer this type over [multikey.Signer] wherever libforge holds a signer.
type Signer struct {
	multikey.Signer
}

var _ multikey.Signer = Signer{}

// NewSigner wraps a multikey signer so that formatting it is safe.
func NewSigner(signer multikey.Signer) Signer {
	return Signer{Signer: signer}
}

// String returns the DID of the signer's key. It never returns private key
// material.
func (s Signer) String() string {
	return s.KeyDID().String()
}
