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
	seen uint64

	// Requested range [start, end) from this reader.
	start, end uint64

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

	timeout time.Duration

	requestQueue chan ReadRequest
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

	itr := &inactiveTimeoutReader{
		object:        object,
		bucket:        bucket,
		start:         byteRange.Start,
		end:           byteRange.Limit,
		parentContext: ctx,
		readHandle:    readHandle,
		mu:            locker.New("inactiveTimeoutReader: "+object.Name, func() {}),
		isActive:      false,
		stopChan:      make(chan struct{}),
		clock:         clock,
		timeout:       timeout,
		requestQueue:  make(chan ReadRequest, 1024),
	}

	// Start the background periodic routine.
	go itr.ProcessRead()

	return itr, nil
}

// createGCSReader is a helper method to create the underlined reader from itr.start + itr.seen offset.
func (itr *inactiveTimeoutReader) createGCSReader(seen uint64) (gcs.StorageReader, error) {
	reader, err := itr.bucket.NewReaderWithReadHandle(
		itr.parentContext,
		&gcs.ReadObjectRequest{
			Name:       itr.object.Name,
			Generation: itr.object.Generation,
			Range: &gcs.ByteRange{
				Start: itr.start + seen,
				Limit: itr.end,
			},
			ReadCompressed: itr.object.HasContentEncodingGzip(),
			ReadHandle:     itr.readHandle,
		})
	if err != nil {
		return nil, fmt.Errorf("NewReaderWithReadHandle: %w", err)
	}
	return reader, nil
}

type Response struct {
	n   int
	err error
}

type ReadRequest struct {
	p            []byte
	responseChan chan<- Response
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
func (itr *inactiveTimeoutReader) Read(p []byte) (n int, err error) {
	ch := make(chan Response)
	itr.requestQueue <- ReadRequest{p: p, responseChan: ch}
	resp := <-ch
	return resp.n, resp.err
}

func (itr *inactiveTimeoutReader) ProcessRead() {
	seen := uint64(0)
	var err error
	for {
		timer := itr.clock.After(itr.timeout)
		select {
		case <-itr.parentContext.Done():
			itr.Close()
			return
		case <-itr.stopChan:
			itr.Close()
			return
		case req, ok := <-itr.requestQueue:
			if !ok {
				break
			}
			if itr.gcsReader == nil {
				itr.gcsReader, err = itr.createGCSReader(seen)
				if err != nil {
					req.responseChan <- Response{
						n:   0,
						err: err,
					}
				}
			}
			n, err := itr.gcsReader.Read(req.p)
			req.responseChan <- Response{
				n:   n,
				err: err,
			}
			seen += uint64(n)
		case <-timer:
			itr.Close()
		}
	}
}

func (itr *inactiveTimeoutReader) Close() error {
	if itr.gcsReader == nil {
		return nil
	}
	itr.readHandle = itr.gcsReader.ReadHandle()
	if err := itr.gcsReader.Close(); err != nil {
		logger.Warnf("Error closing inactive reader for object %q: %v", itr.object.Name, err)
	}
	itr.gcsReader = nil
	return nil
}

// ReadHandle returns the read handle associated with the underlying GCS reader.
// If the reader has been closed due to inactivity, it returns the handle
// stored from the last active reader.
func (itr *inactiveTimeoutReader) ReadHandle() (rh storagev2.ReadHandle) {
	itr.mu.Lock()
	defer itr.mu.Unlock()

	if itr.gcsReader == nil {
		return itr.readHandle
	}

	return itr.gcsReader.ReadHandle()
}
