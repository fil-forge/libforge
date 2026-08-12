package identity_test

import (
	"testing"

	"github.com/fil-forge/libforge/identity"
	"github.com/fil-forge/ucantone/multikey"
	"github.com/fil-forge/ucantone/multikey/ed25519"
	"github.com/stretchr/testify/require"
)

func TestEd25519SignerPEMRoundTrip(t *testing.T) {
	original, err := ed25519.Generate()
	require.NoError(t, err)

	pemBytes, err := identity.EncodeSignerToPEM(original)
	require.NoError(t, err)
	require.NotEmpty(t, pemBytes)

	decoded, err := identity.DecodeSignerFromPEM(pemBytes)
	require.NoError(t, err)

	require.Equal(t, original.Raw(), decoded.Raw())
	require.Equal(t, original.Bytes(), decoded.Bytes())
	require.Equal(t, original.KeyDID(), decoded.KeyDID())
}

// unmarshalableSigner reports a private key that
// ["crypto/x509".MarshalPKCS8PrivateKey] cannot encode, to exercise the error
// path of EncodeSignerToPEM.
type unmarshalableSigner struct {
	multikey.Signer
}

func (s unmarshalableSigner) PrivateKey() any {
	return struct{}{}
}

func TestEncodeSignerToPEM_MarshalErrorNamesSignerByDID(t *testing.T) {
	keySigner, err := ed25519.Generate()
	require.NoError(t, err)

	_, err = identity.EncodeSignerToPEM(unmarshalableSigner{keySigner})

	require.ErrorContains(t, err, "marshaling private key of signer "+keySigner.KeyDID().String())
}

func TestEncodeSignerToPEM_MarshalErrorHidesPrivateKey(t *testing.T) {
	keySigner, err := ed25519.Generate()
	require.NoError(t, err)

	_, err = identity.EncodeSignerToPEM(unmarshalableSigner{keySigner})

	require.NotContains(t, err.Error(), string(keySigner.Raw()))
}

func TestDecodeEd25519SignerFromPEM_NoPrivateKeyBlock(t *testing.T) {
	pemData := []byte("-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n")
	_, err := identity.DecodeSignerFromPEM(pemData)
	require.ErrorContains(t, err, "no PRIVATE KEY block found")
}

func TestDecodeEd25519SignerFromPEM_Empty(t *testing.T) {
	_, err := identity.DecodeSignerFromPEM(nil)
	require.ErrorContains(t, err, "no PRIVATE KEY block found")
}
