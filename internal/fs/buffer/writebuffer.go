package buffer

type WriteBuffer interface {
	// Create creates a buffer of 2*size passed as a parameter.
	Create(size int) WriteBuffer

	// Write writes to the buffer.
	Write(data []byte, offset int64) error

	// Upload asynchronously uploads written data to GCS.
	Upload()

	// Status fetches the stat details of buffer.
	Status()

	// Destroy destorys the buffer when upload to GCS is finalized.
	Destroy()
}
