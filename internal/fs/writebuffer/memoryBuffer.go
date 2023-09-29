package writebuffer

import (
	"bytes"
)

type MemoryBuffer struct {
	// buffer will hold the data to be written to GCS. Buffer holds the data for 2
	// GCS write calls.
	buffer *bytes.Buffer
	// chunkSize is the size of data to be written in one GCS call.
	chunkSize int
	// fileSize is the total data written so far (uploaded to GCS + data in buffer).
	fileSize int
}

func (b *MemoryBuffer) Create(sizeInMB int) WriteBuffer {
	b.chunkSize = sizeInMB * 1024 * 1024
	b.buffer = bytes.NewBuffer(make([]byte, 0, 2*b.chunkSize))
	return b
}

func (b *MemoryBuffer) Write(data []byte, offset int64) {
}

func (b *MemoryBuffer) Upload() {
}

func (b *MemoryBuffer) Status() {
}

func (b *MemoryBuffer) Destroy() {
}
