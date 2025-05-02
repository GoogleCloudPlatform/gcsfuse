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
	"github.com/googlecloudplatform/gcsfuse/v2/internal/clock"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"golang.org/x/net/context"
)

// inactiveTimeoutReader is a wrapper over gcs.StorageReader that automatically
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
type inactiveTimeoutReader struct {
	object *gcs.MinObject
	bucket gcs.Bucket

	// The underlying GCS storage reader; nil if closed due to inactivity.
	gcsReader gcs.StorageReader

	// Total number of bytes successfully read so far.
	seen int64

	// Requested range [start, end) from this reader.
	start, end int64

	// The read handle used for efficient reconnection for zonal bucket.
	readHandle []byte

	// The parent context, used for creating new readers and monitoring cancellation.
	parentContext context.Context

	// Mutex protecting internal state (mainily gcsReader & isActive).
	mu locker.Locker

	// Flag set by Read and reset by the monitor goroutine to track activity within the timeout window.
	isActive bool

	// Channel used to signal the monitor goroutine to stop, typically during Close().
	stopChan chan struct{}

	// Used for waiting for timeout (helps us in mocking the functionality).
	clock clock.Clock
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
func NewInactiveTimeoutReader(ctx context.Context, bucket gcs.Bucket, object *gcs.MinObject, readHandle []byte, start int64, end int64, timeout time.Duration) (gcs.StorageReader, error) {
	if timeout == time.Duration(0) {
		return nil, ErrZeroInactivityTimeout
	}

	return NewInactiveTimeoutReaderWithClock(ctx, bucket, object, readHandle, start, end, timeout, clock.RealClock{})
}

func NewInactiveTimeoutReaderWithClock(ctx context.Context, bucket gcs.Bucket, object *gcs.MinObject, readHandle []byte, start int64, end int64, timeout time.Duration, clock clock.Clock) (gcs.StorageReader, error) {
	if timeout == time.Duration(0) {
		return nil, ErrZeroInactivityTimeout
	}

	tsr := &inactiveTimeoutReader{
		object:        object,
		bucket:        bucket,
		start:         start,
		end:           end,
		parentContext: ctx,
		readHandle:    readHandle,
		mu:            locker.New("inactiveTimeoutReader: "+object.Name, func() {}),
		isActive:      false,
		stopChan:      make(chan struct{}),
		clock:         clock,
	}

	var err error
	tsr.gcsReader, err = tsr.createGCSReader()
	if err == nil {
		go tsr.monitor(timeout)
	}
	return tsr, err
}

// createGCSReader is a helper method to create the underlined reader from tsr.start + tsr.seen offset.
func (tsr *inactiveTimeoutReader) createGCSReader() (gcs.StorageReader, error) {
	reader, err := tsr.bucket.NewReaderWithReadHandle(
		tsr.parentContext,
		&gcs.ReadObjectRequest{
			Name:       tsr.object.Name,
			Generation: tsr.object.Generation,
			Range: &gcs.ByteRange{
				Start: uint64(tsr.start + tsr.seen),
				Limit: uint64(tsr.end),
			},
			ReadCompressed: tsr.object.HasContentEncodingGzip(),
			ReadHandle:     tsr.readHandle,
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
func (tsr *inactiveTimeoutReader) Read(p []byte) (n int, err error) {
	tsr.mu.Lock()
	defer tsr.mu.Unlock()

	tsr.isActive = true

	if tsr.gcsReader == nil {
		tsr.gcsReader, err = tsr.createGCSReader()

		if err != nil {
			return 0, err
		}
	}

	n, err = tsr.gcsReader.Read(p)
	tsr.seen += int64(n)
	return
}

// Close explicitly closes the underlying gcs.StorageReader if it's currently open.
// It also signals the background monitor goroutine to stop.
// Returns an error if closing the underlying reader fails.
func (tsr *inactiveTimeoutReader) Close() (err error) {
	tsr.mu.Lock()
	defer tsr.mu.Unlock()

	close(tsr.stopChan) // Close background monitoring routine.

	if tsr.gcsReader != nil {
		err = tsr.gcsReader.Close()
		tsr.gcsReader = nil
		if err != nil {
			return fmt.Errorf("close reader: %w", err)
		}
	}
	return
}

// ReadHandle returns the read handle associated with the underlying GCS reader.
// If the reader has been closed due to inactivity, it returns the handle
// stored from the last active reader.
func (tsr *inactiveTimeoutReader) ReadHandle() (rh storagev2.ReadHandle) {
	tsr.mu.Lock()
	defer tsr.mu.Unlock()

	if tsr.gcsReader == nil {
		return tsr.readHandle
	}

	return tsr.gcsReader.ReadHandle()
}

// monitor runs in a background goroutine, and checks for inactivity.
func (tsr *inactiveTimeoutReader) monitor(timeout time.Duration) {
	timer := tsr.clock.After(timeout)
	for {
		select {
		case <-timer:
			tsr.mu.Lock()
			if tsr.isActive {
				tsr.isActive = false
				timer = tsr.clock.After(timeout)
			} else {
				if tsr.gcsReader != nil {
					logger.Infof("Closing reader for object %q due to inactivity timeout (%v).\n", tsr.object.Name, timeout)
					tsr.readHandle = tsr.gcsReader.ReadHandle()
					err := tsr.gcsReader.Close()
					if err != nil {
						logger.Warnf("Error closing inactive reader for object %q: %v", tsr.object.Name, err)
					}
					tsr.gcsReader = nil
				}
			}
			tsr.mu.Unlock()
		case <-tsr.parentContext.Done():
			return

		case <-tsr.stopChan:
			return
		}
	}
}
