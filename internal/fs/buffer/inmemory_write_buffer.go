package buffer

import (
	"bytes"
)

type InMemoryWriteBuffer struct {
	// buffer will hold the data to be written to GCS. Buffer holds the data for 2
	// GCS write calls.
	buffer *bytes.Buffer
	// chunkSize is the size of data to be written in one GCS call.
	chunkSize int
}

// NewInMemoryWriteBuffer creates a buffer of 2*sizeInMB passed as a parameter.
func NewInMemoryWriteBuffer(sizeInMB int) *InMemoryWriteBuffer {
	b := &InMemoryWriteBuffer{}
	b.chunkSize = sizeInMB * MiB
	b.buffer = bytes.NewBuffer(make([]byte, 0, 2*b.chunkSize))
	return b
}

func (b *InMemoryWriteBuffer) WriteAt(data []byte, offset int64) error {
	return nil
}
