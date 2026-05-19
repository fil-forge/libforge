//go:build !codegen

package index

import (
	"github.com/fil-forge/libforge/commands"
	"github.com/fil-forge/ucantone/errors"
)

type AddOK = commands.Unit

var Add = commands.MustParse[*AddArguments]("/index/add")

const IndexNotFoundErrorName = "IndexNotFound"

var ErrIndexNotFound = errors.New(IndexNotFoundErrorName, "index not found in space")
