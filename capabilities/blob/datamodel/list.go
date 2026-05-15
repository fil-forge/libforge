package datamodel

type ListArgumentsModel struct {
	Cursor *string `cborgen:"cursor,omitempty" dagjsongen:"cursor,omitempty"`
	Size   *uint64 `cborgen:"size,omitempty" dagjsongen:"size,omitempty"`
}

type ListOKModel struct {
	Cursor  *string        `cborgen:"cursor,omitempty" dagjsongen:"cursor,omitempty"`
	Size    uint64         `cborgen:"size" dagjsongen:"size"`
	Results []ListBlobItem `cborgen:"results" dagjsongen:"results"`
}

type ListBlobItem struct {
	Blob       BlobModel `cborgen:"blob" dagjsongen:"blob"`
	InsertedAt int64     `cborgen:"insertedAt" dagjsongen:"insertedAt"`
}
