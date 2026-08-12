package identity_test

import (
	"fmt"
	"testing"

	"github.com/fil-forge/libforge/identity"
	"github.com/fil-forge/ucantone/multikey/ed25519"
	"github.com/stretchr/testify/require"
)

func TestSignerStringReturnsKeyDID(t *testing.T) {
	keySigner, err := ed25519.Generate()
	require.NoError(t, err)

	signer := identity.NewSigner(keySigner)

	require.Equal(t, keySigner.KeyDID().String(), signer.String())
}

// Formatting verbs a signer may plausibly reach in an error message or a log
// line. None of them may print the private key.
var signerFormatVerbs = map[string]string{
	"%s": "%s",
	"%v": "%s",
	"%q": "%q",
}

func TestSignerFormattingPrintsKeyDID(t *testing.T) {
	for verb, expectedVerb := range signerFormatVerbs {
		t.Run(verb, func(t *testing.T) {
			keySigner, err := ed25519.Generate()
			require.NoError(t, err)

			formatted := fmt.Sprintf(verb, identity.NewSigner(keySigner))

			expected := fmt.Sprintf(expectedVerb, keySigner.KeyDID().String())
			require.Equal(t, expected, formatted)
		})
	}
}
