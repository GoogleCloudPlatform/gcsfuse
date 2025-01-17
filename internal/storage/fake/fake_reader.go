package fake

import (
	"io"

	storagev2 "cloud.google.com/go/storage"
)

// FakeReader implements gcs.StorageReader interface
type FakeReader struct {
	io.ReadCloser
	Handle []byte
}

func (fr *FakeReader) ReadHandle() storagev2.ReadHandle {
	return fr.Handle
}
