// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcsx

import (
	"errors"
	"fmt"
	"time"

	storagev2 "cloud.google.com/go/storage"
	"github.com/vipnydav/gcsfuse/v3/internal/clock"
	"github.com/vipnydav/gcsfuse/v3/internal/locker"
	"github.com/vipnydav/gcsfuse/v3/internal/logger"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/gcs"
	"golang.org/x/net/context"
)

// InactiveTimeoutReader is a wrapper over gcs.StorageReader that automatically
// closes the wrapped GCS reader connection after a specified period of
// inactivity (timeout). When a read operation is attempted on an inactive
// (closed) reader, it automatically attempts to reconnect using the last known
// read handle and the appropriate offset based on bytes previously read.
//
// This is useful for managing resources, especially when dealing with many
// potentially inactive or idle readers.
//
// Important notes:
//   - Inactivity Timer: A background goroutine monitors read activity. If no
//     Read calls occur within the timeout duration, the underlying gcsReader is
//     closed.
//   - Due to the activity check happens periodically (every timeout duration), the
//     actual reader connection closure can happen anywhere b/w timeout and 2 * timeout
//     after the very last read operation, depending on when the last read occurred
//     relative to the background routine wake-up cycle.
//   - Thread Safety: The reader is safe for concurrent use by multiple goroutines,
//     protected by an internal mutex.
type InactiveTimeoutReader struct {
	object *gcs.MinObject
	bucket gcs.Bucket

	// The underlying GCS storage reader; nil if closed due to inactivity.
	gcsReader gcs.StorageReader

	// Total number of bytes successfully read so far.
	seen uint64

	// Requested range [start, end) from this reader.
	reqRange gcs.ByteRange

	// The read handle used for efficient reconnection for zonal bucket.
	readHandle []byte

	// Derived from the parent context, used for creating new readers and monitoring cancellation.
	ctx    context.Context
	cancel context.CancelFunc

	// Mutex protecting internal state (mainly gcsReader & isActive).
	mu locker.Locker

	// Flag set by Read and reset by the monitor goroutine to track activity within the timeout window.
	isActive bool
}

var (
	ErrZeroInactivityTimeout = errors.New("ErrZeroInactivityTimeout")
)

// NewInactiveTimeoutReader creates a new gcs.StorageReader that wraps an
// underlying GCS reader. It attempts to create the initial reader using the
// provided parameters. If successful, it starts a background goroutine to monitor
// for inactivity based on the specified timeout.
//
// If the timeout duration is zero, it returns (nil, ErrZeroInactivityTimeout) as a zero timeout
// defeats the purpose of this wrapper.
func NewInactiveTimeoutReader(ctx context.Context, bucket gcs.Bucket, object *gcs.MinObject, readHandle []byte, byteRange gcs.ByteRange, timeout time.Duration) (gcs.StorageReader, error) {
	return NewInactiveTimeoutReaderWithClock(ctx, bucket, object, readHandle, byteRange, timeout, clock.RealClock{})
}

func NewInactiveTimeoutReaderWithClock(ctx context.Context, bucket gcs.Bucket, object *gcs.MinObject, readHandle []byte, byteRange gcs.ByteRange, timeout time.Duration, clock clock.Clock) (gcs.StorageReader, error) {
	if timeout == 0 {
		return nil, ErrZeroInactivityTimeout
	}

	itr := &InactiveTimeoutReader{
		object:     object,
		bucket:     bucket,
		reqRange:   byteRange,
		readHandle: readHandle,
		mu:         locker.New("InactiveTimeoutReader: "+object.Name, func() {}),
		isActive:   false,
	}
	itr.ctx, itr.cancel = context.WithCancel(ctx)

	var err error
	if itr.gcsReader, err = itr.createGCSReader(); err != nil {
		return nil, err
	}

	// Start the background periodic routine.
	go itr.monitor(clock, timeout)

	return itr, nil
}

// createGCSReader is a helper method to create the underlined reader from itr.start + itr.seen offset.
func (itr *InactiveTimeoutReader) createGCSReader() (gcs.StorageReader, error) {
	reader, err := itr.bucket.NewReaderWithReadHandle(
		itr.ctx,
		&gcs.ReadObjectRequest{
			Name:       itr.object.Name,
			Generation: itr.object.Generation,
			Range: &gcs.ByteRange{
				Start: itr.reqRange.Start + itr.seen,
				Limit: itr.reqRange.Limit,
			},
			ReadCompressed: itr.object.HasContentEncodingGzip(),
			ReadHandle:     itr.readHandle,
		})
	if err != nil {
		return nil, fmt.Errorf("NewReaderWithReadHandle: %w", err)
	}
	return reader, nil
}

// Read implements io.Reader interface.
//
// If the underlying reader has been closed due to inactivity, Read automatically
// attempts to reconnect using the stored read handle and the correct offset
// (start + bytes previously seen). If reconnection fails, the error is returned.
//
// Each successful Read call marks the reader as active, resetting the inactivity timer
// in the background monitor. This method is thread-safe.
//
// Calling Read() after explicitly calling Close() is not supported and will
// lead to undefined behavior.
func (itr *InactiveTimeoutReader) Read(p []byte) (n int, err error) {
	itr.mu.Lock()
	defer itr.mu.Unlock()

	itr.isActive = true

	if itr.gcsReader == nil {
		if itr.gcsReader, err = itr.createGCSReader(); err != nil {
			return 0, err
		}
	}

	n, err = itr.gcsReader.Read(p)
	itr.seen += uint64(n)
	return n, err
}

// Close explicitly closes the underlying gcs.StorageReader if it's currently open.
// It also signals the background monitor goroutine to stop.
// Returns an error if closing the underlying reader fails.
func (itr *InactiveTimeoutReader) Close() (err error) {
	itr.mu.Lock()
	defer itr.mu.Unlock()

	// Signal background periodic routine to stop.
	itr.cancel()

	if itr.gcsReader == nil {
		return nil
	}

	err = itr.gcsReader.Close()
	itr.gcsReader = nil
	if err != nil {
		return fmt.Errorf("Close reader: %w", err)
	}
	return nil
}

// ReadHandle returns the read handle associated with the underlying GCS reader.
// If the reader has been closed due to inactivity, it returns the handle
// stored from the last active reader.
func (itr *InactiveTimeoutReader) ReadHandle() (rh storagev2.ReadHandle) {
	itr.mu.Lock()
	defer itr.mu.Unlock()

	if itr.gcsReader == nil {
		return itr.readHandle
	}

	return itr.gcsReader.ReadHandle()
}

// monitor runs in a background goroutine, and checks for inactivity.
func (itr *InactiveTimeoutReader) monitor(clock clock.Clock, timeout time.Duration) {
	timer := clock.After(timeout)
	for {
		select {
		case <-timer:
			itr.handleTimeout()
			timer = clock.After(timeout)
		case <-itr.ctx.Done():
			return
		}
	}
}

// handleTimeout is called when the inactivity timer fires. It acquires the
// reader's lock, checks the activity state, and takes appropriate action.
// If the reader was marked as active since the last check, it resets the
// activity flag. If the reader was inactive, it closes the underlying GCS reader.
// It always returns a new timer channel for the next fire.
func (itr *InactiveTimeoutReader) handleTimeout() {
	itr.mu.Lock()
	defer itr.mu.Unlock()

	if itr.isActive {
		itr.isActive = false
	} else {
		itr.closeGCSReader()
	}
}

// closeGCSReader closes the wrapped gcsReader, itr.mu.Lock() should be taken
// before calling this.
func (itr *InactiveTimeoutReader) closeGCSReader() {
	if itr.gcsReader == nil {
		return
	}

	// Not printing the timeout explicitly, as can be refer from the code/config.
	logger.Tracef("Closing reader for object %q due to inactivity.\n", itr.object.Name)
	itr.readHandle = itr.gcsReader.ReadHandle()
	if err := itr.gcsReader.Close(); err != nil {
		logger.Warnf("Error closing inactive reader for object %q: %v", itr.object.Name, err)
	}
	itr.gcsReader = nil
}
