//go:build !codegen

package ucan

import (
	"github.com/fil-forge/libforge/capabilities"
	"github.com/fil-forge/ucantone/errors"
)

const ConcludeCommand = "/ucan/conclude"

type ConcludeOK = capabilities.Unit

var Conclude = capabilities.MustNew[*ConcludeArguments](ConcludeCommand)

const ConclusionReceiptNotFoundErrorName = "ConclusionReceiptNotFound"

var ErrConclusionReceiptNotFound = errors.New(ConclusionReceiptNotFoundErrorName, "conclusion receipt not found")
