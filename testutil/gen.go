package testutil

import (
	"github.com/fil-forge/ucantone/testutil"
)

var (
	RandomBytes          = testutil.RandomBytes
	RandomCID            = testutil.RandomCID
	RandomDigest         = testutil.RandomDigest
	RandomDID            = testutil.RandomDID
	RandomSigner         = testutil.RandomSigner
	RandomIssuer         = testutil.RandomIssuer
	RandomMultikeySigner = testutil.RandomMultikeySigner
	RandomMultikeyIssuer = testutil.RandomMultikeyIssuer
	RandomPrincipal      = testutil.RandomPrincipal

	// Deprecated alias for RandomDigest, which is a more accurate name.
	RandomMultihash = testutil.RandomDigest
)
