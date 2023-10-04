package buffer

const (
	// MiB is the multiplication factor to convert MiB to bytes.
	MiB = 1024 * 1024
	// InMemoryBufferThresholdMB is the upper limit on the size upto which the buffer should
	// be created in memory. Beyond this size, buffer should be on disk.
	InMemoryBufferThresholdMB = 50
)

// WriteBuffer is an interface that buffers the data to be written to GCS during
// the write flow.
// WriteBuffer is used only in create new file flow with sequential writes and
// at any point in time, only 2x of the configured buffer size is stored in the
// write buffer.
type WriteBuffer interface {
	// WriteAt writes at an offset to the buffer.
	WriteAt(data []byte, offset int64) error
}
