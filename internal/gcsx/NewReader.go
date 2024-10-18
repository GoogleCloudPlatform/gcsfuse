package gcsx

import (
	"context"
	"fmt"
	"io"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
)

type NewReader struct {
	object *gcs.MinObject
	bucket gcs.Bucket

	// If non-nil, an in-flight read request and a function for cancelling it.
	//
	// INVARIANT: (reader == nil) == (cancel == nil)
	reader io.ReadCloser
	cancel func()

	// The range of the object that we expect reader to yield, when reader is
	// non-nil. When reader is nil, limit is the limit of the previous read
	// operation, or -1 if there has never been one.
	//
	// INVARIANT: start <= limit
	// INVARIANT: limit < 0 implies reader != nil
	// All these properties will be used only in case of GCS reads and not for
	// reads from cache.
	start          int64
	limit          int64
	seeks          uint64
	totalReadBytes uint64
}

func (nr *NewReader) TryServingFromNewReader(p []byte, offset int64, len int64) (bool, []byte) {
	// No existing connection available to serve data.
	if nr.reader == nil {
		return false, nil
	}

	// nr.limit <= offset+len - This is a new condition to check if we can serve complete data using existing reader..
	if nr.start < offset && nr.limit <= offset+len && offset-nr.start < maxReadSize {
		bytesToSkip := int64(offset - nr.start)
		p := make([]byte, bytesToSkip)
		n, _ := io.ReadFull(nr.reader, p)
		nr.start += int64(n)
	}
	// Other logic to read data and update all variables. We can reuse from Read method below

	return true, p

}

// End tells what is the end offset when network call is made. Size is the actual size of data requested.
func (nr *NewReader) Read(ctx context.Context, p []byte, offset int64, end int64, size int64) (output []byte, err error) {
	for size > 0 {
		// Have we blown past the end of the object?
		if offset >= int64(nr.object.Size) {
			err = io.EOF
			return
		}

		// When the offset is AFTER the reader position, try to seek forward, within reason.
		// This happens when the kernel page cache serves some data. It's very common for
		// concurrent reads, often by only a few 128kB fuse read requests. The aim is to
		// re-use GCS connection and avoid throwing away already read data.
		// For parallel sequential reads to a single file, not throwing away the connections
		// is a 15-20x improvement in throughput: 150-200 MB/s instead of 10 MB/s.
		if nr.reader != nil && nr.start < offset && offset-nr.start < maxReadSize {
			bytesToSkip := int64(offset - nr.start)
			p := make([]byte, bytesToSkip)
			n, _ := io.ReadFull(nr.reader, p)
			nr.start += int64(n)
		}

		// If we have an existing reader but it's positioned at the wrong place,
		// clean it up and throw it away.
		if nr.reader != nil && nr.start != offset {
			nr.reader.Close()
			nr.reader = nil
			nr.cancel = nil
			nr.seeks++
		}

		// If we don't have a reader, start a read operation.
		if nr.reader == nil {

			err = nr.startRead(ctx, offset, end)
			if err != nil {
				err = fmt.Errorf("startRead: %w", err)
				return
			}
		}

		// Now we have a reader positioned at the correct place. Consume as much from
		// it as possible.
		var tmp int
		tmp, err = nr.readFull(ctx, p)

		n += tmp
		p = p[tmp:]
		nr.start += int64(tmp)
		offset += int64(tmp)
		nr.totalReadBytes += uint64(tmp)

		// Sanity check.
		if nr.start > nr.limit {
			err = fmt.Errorf("reader returned %d too many bytes", rr.start-rr.limit)

			// Don't attempt to reuse the reader when it's behaving wackily.
			nr.reader.Close()
			nr.reader = nil
			nr.cancel = nil
			nr.start = -1
			nr.limit = -1

			return
		}

		// Are we finished with this reader now?
		if nr.start == nr.limit {
			nr.reader.Close()
			nr.reader = nil
			nr.cancel = nil
		}

		// Handle errors.
		switch {
		case err == io.EOF || err == io.ErrUnexpectedEOF:
			// For a non-empty buffer, ReadFull returns EOF or ErrUnexpectedEOF only
			// if the reader peters out early. That's fine, but it means we should
			// have hit the limit above.
			if nr.reader != nil {
				err = fmt.Errorf("reader returned %d too few bytes", rr.limit-rr.start)
				return
			}

			err = nil

		case err != nil:
			// Propagate other errors.
			err = fmt.Errorf("readFull: %w", err)
			return
		}
	}

	return

}

func (nr *NewReader) startRead(ctx context.Context, start int64, end int64) error {
	ctx, cancel := context.WithCancel(context.Background())
	rc, err := nr.bucket.NewReader(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       nr.object.Name,
			Generation: nr.object.Generation,
			Range: &gcs.ByteRange{
				Start: uint64(start),
				Limit: uint64(end),
			},
			ReadCompressed: nr.object.HasContentEncodingGzip(),
		})

	if err != nil {
		err = fmt.Errorf("NewReader: %w", err)
		return err
	}

	nr.reader = rc
	nr.cancel = cancel
	nr.start = start
	nr.limit = end

	requestedDataSize := end - start
	monitor.CaptureGCSReadMetrics(ctx, util.Sequential, requestedDataSize)
	return nil
}

// Copied from random_reader.go. No changes in code.
// Like io.ReadFull, but deals with the cancellation issues.
//
// REQUIRES: rr.reader != nil
func (rr *randomReader) readFull(
	ctx context.Context,
	p []byte) (n int, err error) {
	// Start a goroutine that will cancel the read operation we block on below if
	// the calling context is cancelled, but only if this method has not already
	// returned (to avoid souring the reader for the next read if this one is
	// successful, since the calling context will eventually be cancelled).
	readDone := make(chan struct{})
	defer close(readDone)

	go func() {
		select {
		case <-readDone:
			return

		case <-ctx.Done():
			select {
			case <-readDone:
				return

			default:
				rr.cancel()
			}
		}
	}()

	// Call through.
	n, err = io.ReadFull(rr.reader, p)

	return
}
