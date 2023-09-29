package buffer

import "io"

// WriteBuffer is an interface that buffers the data to be written to GCS during
// the write flow.
// WriteBuffer is used only in create new file flow with sequential writes and
// at any point in time, only 2x of the configured buffer size is stored in the
// write buffer.
type WriteBuffer interface {
	io.Reader
	// Create creates a buffer of 2*size passed as a parameter.
	Create(size int) WriteBuffer

	// Write writes to the buffer.
	Write(data []byte, offset int64) error

	// Upload asynchronously uploads written data to GCS.
	Upload()

	// Status fetches the stat details of buffer.
	Status()

	// Destroy destroys the buffer when upload to GCS is finalized.
	Destroy()
}
