package writebuffer

import (
	"bytes"

	"github.com/googlecloudplatform/gcsfuse/internal/logger"
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
	logger.Infof("In memory buffer created of size %v, %T", sizeInMB, *b)
	return b
}

func (b *MemoryBuffer) Write(data []byte, offset int64) {
	b.buffer.Write(data)
	if b.buffer.Len() <= b.chunkSize {
		b.Upload()
	}
}

func (b *MemoryBuffer) Upload() {
	logger.Info("upload trigerred")
}

func (b *MemoryBuffer) Status() {
}

func (b *MemoryBuffer) Destroy() {
}
