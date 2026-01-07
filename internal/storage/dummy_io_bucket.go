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

package storage

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

const MB = 1024 * 1024

type DummyIOBucketParams struct {
	ReaderLatency time.Duration
	PerMBLatency  time.Duration
}

// dummyIOBucket is a wrapper over gcs.Bucket that implements gcs.Bucket interface.
// It embeds baseBucketWrapper to inherit default delegation behavior, and only
// overrides the read operations to perform dummy IO for testing purposes.
type dummyIOBucket struct {
	baseBucketWrapper
	readerLatency time.Duration
	perMBLatency  time.Duration
}

// NewDummyIOBucket creates a new dummyIOBucket wrapping the given gcs.Bucket.
// If the wrapped bucket is nil, it returns nil.
func NewDummyIOBucket(wrapped gcs.Bucket, params DummyIOBucketParams) gcs.Bucket {
	if wrapped == nil {
		return nil
	}

	return &dummyIOBucket{
		baseBucketWrapper: baseBucketWrapper{wrapped: wrapped},
		readerLatency:     params.ReaderLatency,
		perMBLatency:      params.PerMBLatency,
	}
}

// NewReaderWithReadHandle creates a reader for reading object contents.
// Returns a dummy reader that serves zeros efficiently instead of reading from GCS.
func (d *dummyIOBucket) NewReaderWithReadHandle(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (gcs.StorageReader, error) {

	if req.Range == nil {
		return nil, errors.New("range must be specified for dummy IO bucket")
	}

	rangeLen := int64(req.Range.Limit) - int64(req.Range.Start)
	if rangeLen <= 0 {
		return nil, errors.New("invalid range: limit is less than start")
	}

	// Simulate network latency if specified.
	if d.readerLatency > 0 {
		time.Sleep(d.readerLatency)
	}

	return newDummyReader(uint64(rangeLen), d.perMBLatency), nil
}

// NewMultiRangeDownloader creates a multi-range downloader for object contents.
func (d *dummyIOBucket) NewMultiRangeDownloader(
	ctx context.Context,
	req *gcs.MultiRangeDownloaderRequest) (gcs.MultiRangeDownloader, error) {
	return &dummyMultiRangeDownloader{perMBLatency: d.perMBLatency}, nil
}

// All other methods (CreateObject, DeleteFolder, GetFolder, etc.) are inherited
// from baseBucketWrapper and delegate directly to the wrapped bucket.
// This eliminates ~170 lines of boilerplate delegation code.

////////////////////////////////////////////////////////////////////////
// dummyReader
////////////////////////////////////////////////////////////////////////

// dummyReader is an efficient reader that serves dummy data.
// Reading beyond the specified length returns io.EOF.
// Also, it always returns a non-nil read handle.
type dummyReader struct {
	totalLen     uint64 // Total length of data to serve
	bytesRead    uint64 // Number of bytes already read
	readHandle   storagev2.ReadHandle
	perMBLatency time.Duration
}

func calculateLatency(bytes int64, perMBLatency time.Duration) time.Duration {
	if perMBLatency <= 0 {
		return 0
	}
	return time.Duration(float64(bytes) * float64(perMBLatency.Nanoseconds()) / float64(MB))
}

// newDummyReader creates a new dummyReader with the specified total length.
func newDummyReader(totalLen uint64, perMBLatency time.Duration) *dummyReader {
	return &dummyReader{
		totalLen:     totalLen,
		bytesRead:    0,
		readHandle:   []byte{}, // Always return a non-nil read handle
		perMBLatency: perMBLatency,
	}
}

// Read reads up to len(p) bytes into p, filling it with zeros.
// Returns io.EOF when the total length has been reached.
func (dr *dummyReader) Read(p []byte) (n int, err error) {
	// If we've already read all the data, return EOF
	if dr.bytesRead >= dr.totalLen {
		return 0, io.EOF
	}

	// Calculate how many bytes we can still read
	remaining := dr.totalLen - dr.bytesRead

	// Determine how many bytes to read in this call
	toRead := uint64(len(p))
	if toRead > remaining {
		toRead = remaining
	}

	// Simulate per-MB latency if specified
	time.Sleep(calculateLatency(int64(toRead), dr.perMBLatency))

	dr.bytesRead += toRead

	// If we've read all the data, return EOF along with the last bytes
	if dr.bytesRead >= dr.totalLen {
		return int(toRead), io.EOF
	}

	return int(toRead), nil
}

// Close closes the reader. For dummy reader, this is a no-op.
func (dr *dummyReader) Close() error {
	return nil
}

// ReadHandle returns the read handle. For dummy reader, this returns a nil handle.
func (dr *dummyReader) ReadHandle() storagev2.ReadHandle {
	return dr.readHandle
}

////////////////////////////////////////////////////////////////////////
// dummyMultiRangeDownloader
////////////////////////////////////////////////////////////////////////

type dummyMultiRangeDownloader struct {
	perMBLatency time.Duration
	wg           sync.WaitGroup
}

// zeroReader is an io.Reader that always reads zeros.
type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func (d *dummyMultiRangeDownloader) Add(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()

		// Simulate latency
		time.Sleep(calculateLatency(length, d.perMBLatency))

		// Write zeros
		// output writer is bytes.Buffer which implements io.ReaderFrom interface
		bytesWritten, err := io.Copy(output, io.LimitReader(zeroReader{}, length))

		if callback != nil {
			callback(offset, bytesWritten, err)
		}
	}()
}

func (d *dummyMultiRangeDownloader) Close() error {
	d.Wait()
	return nil
}

func (d *dummyMultiRangeDownloader) Wait() {
	d.wg.Wait()
}

func (d *dummyMultiRangeDownloader) Error() error {
	return nil
}

func (d *dummyMultiRangeDownloader) GetHandle() []byte {
	return []byte("dummy-handle")
}
