package identity

import (
	"bytes"
	crypto_ed25519 "crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/fil-forge/ucantone/multikey"
	"github.com/fil-forge/ucantone/multikey/ed25519"
)

// EncodeSignerToPEM encodes a signer to a PKCS#8 PEM format. The signer's key
// should be of a type supported by ["crypto/x509".MarshalPKCS8PrivateKey].
func EncodeSignerToPEM(keySigner multikey.Signer) ([]byte, error) {
	// Wrap the signer so the error messages below name it by its DID instead of
	// printing its private key bytes.
	signer := NewSigner(keySigner)

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(signer.PrivateKey())
	if err != nil {
		return nil, fmt.Errorf("marshaling private key of signer %s: %w", signer, err)
	}

	privateKeyBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	buffer := new(bytes.Buffer)
	if err := pem.Encode(buffer, privateKeyBlock); err != nil {
		return nil, fmt.Errorf("encoding private key of signer %s: %w", signer, err)
	}

	return buffer.Bytes(), nil
}

// DecodeSignerFromPEM loads a private key from a PKCS#8 PEM as a signer.
// Currently, only Ed25519 keys are supported.
func DecodeSignerFromPEM(pemData []byte) (Signer, error) {
	var privateKey *crypto_ed25519.PrivateKey
	rest := pemData
	for {
		block, remaining := pem.Decode(rest)
		if block == nil {
			break
		}
		rest = remaining

		if block.Type == "PRIVATE KEY" {
			parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return Signer{}, fmt.Errorf("parsing PKCS#8 private key: %w", err)
			}

			key, ok := parsedKey.(crypto_ed25519.PrivateKey)
			if !ok {
				return Signer{}, fmt.Errorf("key is not an Ed25519 private key")
			}
			privateKey = &key
			break
		}
	}

	if privateKey == nil {
		return Signer{}, fmt.Errorf("no PRIVATE KEY block found in PEM file")
	}

	signer, err := ed25519.FromRaw(privateKey.Seed())
	if err != nil {
		return Signer{}, fmt.Errorf("creating signer from private key: %w", err)
	}

	return NewSigner(signer), nil
}
