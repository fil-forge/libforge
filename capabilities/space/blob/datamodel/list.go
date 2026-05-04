package datamodel

import sdm "github.com/alanshaw/libracha/capabilities/shared/datamodel"

type ListArgumentsModel struct {
	Cursor *string `cborgen:"cursor,omitempty" dagjsongen:"cursor,omitempty"`
	Size   *uint64 `cborgen:"size,omitempty" dagjsongen:"size,omitempty"`
}

type ListOKModel struct {
	Cursor  *string        `cborgen:"cursor,omitempty" dagjsongen:"cursor,omitempty"`
	Before  *string        `cborgen:"before,omitempty" dagjsongen:"before,omitempty"`
	After   *string        `cborgen:"after,omitempty" dagjsongen:"after,omitempty"`
	Size    uint64         `cborgen:"size" dagjsongen:"size"`
	Results []ListBlobItem `cborgen:"results" dagjsongen:"results"`
}

type ListBlobItem struct {
	Blob       sdm.BlobModel `cborgen:"blob" dagjsongen:"blob"`
	InsertedAt uint64        `cborgen:"insertedAt" dagjsongen:"insertedAt"`
}
