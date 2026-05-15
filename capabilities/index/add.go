//go:build !codegen

package index

import (
	"github.com/fil-forge/libforge/capabilities"
	"github.com/fil-forge/ucantone/errors"
)

const AddCommand = "/index/add"

type AddOK = capabilities.Unit

var Add = capabilities.MustNew[*AddArguments](AddCommand)

const IndexNotFoundErrorName = "IndexNotFound"

var ErrIndexNotFound = errors.New(IndexNotFoundErrorName, "index not found in space")
