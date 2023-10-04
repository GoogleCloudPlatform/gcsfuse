package buffer

// MiB is the multiplication factor to convert MiB to bytes.
const MiB = 1024 * 1024

// WriteBuffer is an interface that buffers the data to be written to GCS during
// the write flow.
// WriteBuffer is used only in create new file flow with sequential writes and
// at any point in time, only 2x of the configured buffer size is stored in the
// write buffer.
type WriteBuffer interface {
	// WriteAt writes at an offset to the buffer.
	WriteAt(data []byte, offset int64) error
}
