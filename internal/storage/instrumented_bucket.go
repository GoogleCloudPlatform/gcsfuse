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
	"io"
	"sync/atomic"
	"time"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/benchmark"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// InstrumentedBucketParams configures an instrumentedBucket.
// The bucket name is used to tag events; trackName identifies which benchmark
// track is driving I/O through this wrapper.
type InstrumentedBucketParams struct {
	// TrackName is the benchmark track label applied to all events.
	TrackName string

	// Hists is the HDR histogram store that receives per-event latency samples.
	Hists *benchmark.TrackHistograms

	// Events receives raw PerfEvent values for post-processing (may be nil).
	Events chan<- benchmark.PerfEvent
}

// instrumentedBucket wraps gcs.Bucket and records I/O latency into HDR
// histograms. All non-I/O methods are passed through unchanged.
type instrumentedBucket struct {
	wrapped   gcs.Bucket
	trackName string
	hists     *benchmark.TrackHistograms
	events    chan<- benchmark.PerfEvent

	// Counters used for throughput calculation.
	totalBytes atomic.Int64
	totalOps   atomic.Int64
	totalErrs  atomic.Int64
}

// NewInstrumentedBucket wraps a gcs.Bucket with latency instrumentation.
// Returns nil if wrapped is nil.
func NewInstrumentedBucket(wrapped gcs.Bucket, params InstrumentedBucketParams) gcs.Bucket {
	if wrapped == nil {
		return nil
	}
	return &instrumentedBucket{
		wrapped:   wrapped,
		trackName: params.TrackName,
		hists:     params.Hists,
		events:    params.Events,
	}
}

func (b *instrumentedBucket) Name() string {
	return b.wrapped.Name()
}

func (b *instrumentedBucket) BucketType() gcs.BucketType {
	return b.wrapped.BucketType()
}

// NewReaderWithReadHandle wraps the underlying reader with an instrumentedReader
// that records TTFB on the first Read() call and total latency on Close().
func (b *instrumentedBucket) NewReaderWithReadHandle(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (gcs.StorageReader, error) {

	start := time.Now()
	logger.Tracef("[gcs] READ start: %s\n", req.Name)
	r, err := b.wrapped.NewReaderWithReadHandle(ctx, req)
	if err != nil {
		b.totalErrs.Add(1)
		logger.Debugf("[gcs] READ error: %s: %v\n", req.Name, err)
		b.recordEvent(benchmark.PerfEvent{
			Op:           benchmark.OpRead,
			TotalLatency: time.Since(start),
			Err:          err,
		})
		return nil, err
	}

	// Determine the object size hint from the request range; fall back to 0.
	var objectSize int64
	if req.Range != nil {
		objectSize = int64(req.Range.Limit) - int64(req.Range.Start)
		if objectSize < 0 {
			objectSize = 0
		}
	}

	return &instrumentedReader{
		StorageReader: r,
		bucket:        b,
		start:         start,
		objectSize:    objectSize,
	}, nil
}

// NewMultiRangeDownloader passes through — multirange is not benchmarked here.
func (b *instrumentedBucket) NewMultiRangeDownloader(
	ctx context.Context,
	req *gcs.MultiRangeDownloaderRequest) (gcs.MultiRangeDownloader, error) {
	return b.wrapped.NewMultiRangeDownloader(ctx, req)
}

// CreateObject instruments write latency.
func (b *instrumentedBucket) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (*gcs.Object, error) {

	start := time.Now()
	logger.Tracef("[gcs] WRITE start: %s\n", req.Name)
	obj, err := b.wrapped.CreateObject(ctx, req)
	elapsed := time.Since(start)

	b.totalOps.Add(1)
	if err != nil {
		b.totalErrs.Add(1)
		logger.Debugf("[gcs] WRITE error: %s: %v\n", req.Name, err)
		b.recordEvent(benchmark.PerfEvent{Op: benchmark.OpWrite, TotalLatency: elapsed, Err: err})
		return nil, err
	}

	var size int64
	if obj != nil {
		size = int64(obj.Size)
		b.totalBytes.Add(size)
	}
	logger.Tracef("[gcs] WRITE done: %s size=%d elapsed=%s\n", req.Name, size, elapsed.Round(time.Millisecond))
	b.hists.RecordTotal(elapsed.Microseconds())
	b.recordEvent(benchmark.PerfEvent{
		Op:               benchmark.OpWrite,
		TotalLatency:     elapsed,
		BytesTransferred: size,
		ObjectSize:       size,
	})
	return obj, nil
}

// CreateObjectChunkWriter passes through.
func (b *instrumentedBucket) CreateObjectChunkWriter(
	ctx context.Context,
	req *gcs.CreateObjectRequest,
	chunkSize int,
	callBack func(bytesUploadedSoFar int64)) (gcs.Writer, error) {
	return b.wrapped.CreateObjectChunkWriter(ctx, req, chunkSize, callBack)
}

// CreateAppendableObjectWriter passes through.
func (b *instrumentedBucket) CreateAppendableObjectWriter(
	ctx context.Context,
	req *gcs.CreateObjectChunkWriterRequest) (gcs.Writer, error) {
	return b.wrapped.CreateAppendableObjectWriter(ctx, req)
}

// FinalizeUpload passes through.
func (b *instrumentedBucket) FinalizeUpload(
	ctx context.Context,
	writer gcs.Writer) (*gcs.MinObject, error) {
	return b.wrapped.FinalizeUpload(ctx, writer)
}

// FlushPendingWrites passes through.
func (b *instrumentedBucket) FlushPendingWrites(
	ctx context.Context,
	writer gcs.Writer) (*gcs.MinObject, error) {
	return b.wrapped.FlushPendingWrites(ctx, writer)
}

// CopyObject passes through.
func (b *instrumentedBucket) CopyObject(
	ctx context.Context,
	req *gcs.CopyObjectRequest) (*gcs.Object, error) {
	return b.wrapped.CopyObject(ctx, req)
}

// ComposeObjects passes through.
func (b *instrumentedBucket) ComposeObjects(
	ctx context.Context,
	req *gcs.ComposeObjectsRequest) (*gcs.Object, error) {
	return b.wrapped.ComposeObjects(ctx, req)
}

// StatObject instruments stat latency.
func (b *instrumentedBucket) StatObject(
	ctx context.Context,
	req *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {

	start := time.Now()
	logger.Tracef("[gcs] STAT start: %s\n", req.Name)
	mo, ext, err := b.wrapped.StatObject(ctx, req)
	elapsed := time.Since(start)

	b.totalOps.Add(1)
	if err != nil {
		b.totalErrs.Add(1)
		logger.Debugf("[gcs] STAT error: %s: %v\n", req.Name, err)
		b.recordEvent(benchmark.PerfEvent{Op: benchmark.OpStat, TotalLatency: elapsed, Err: err})
		return nil, nil, err
	}
	logger.Tracef("[gcs] STAT done: %s elapsed=%s\n", req.Name, elapsed.Round(time.Millisecond))
	b.hists.RecordTotal(elapsed.Microseconds())
	b.recordEvent(benchmark.PerfEvent{Op: benchmark.OpStat, TotalLatency: elapsed})
	return mo, ext, nil
}

// ListObjects instruments list latency.
func (b *instrumentedBucket) ListObjects(
	ctx context.Context,
	req *gcs.ListObjectsRequest) (*gcs.Listing, error) {

	start := time.Now()
	logger.Tracef("[gcs] LIST start: prefix=%q\\n", req.Prefix)
	listing, err := b.wrapped.ListObjects(ctx, req)
	elapsed := time.Since(start)

	b.totalOps.Add(1)
	if err != nil {
		b.totalErrs.Add(1)
		logger.Debugf("[gcs] LIST error: prefix=%q: %v\\n", req.Prefix, err)
		b.recordEvent(benchmark.PerfEvent{Op: benchmark.OpList, TotalLatency: elapsed, Err: err})
		return nil, err
	}
	count := 0
	if listing != nil {
		count = len(listing.MinObjects)
	}
	logger.Tracef("[gcs] LIST done: prefix=%q count=%d elapsed=%s\\n", req.Prefix, count, elapsed.Round(time.Millisecond))
	b.hists.RecordTotal(elapsed.Microseconds())
	b.recordEvent(benchmark.PerfEvent{Op: benchmark.OpList, TotalLatency: elapsed})
	return listing, nil
}

// UpdateObject passes through.
func (b *instrumentedBucket) UpdateObject(
	ctx context.Context,
	req *gcs.UpdateObjectRequest) (*gcs.Object, error) {
	return b.wrapped.UpdateObject(ctx, req)
}

// DeleteObject passes through.
func (b *instrumentedBucket) DeleteObject(
	ctx context.Context,
	req *gcs.DeleteObjectRequest) error {
	return b.wrapped.DeleteObject(ctx, req)
}

// MoveObject passes through.
func (b *instrumentedBucket) MoveObject(
	ctx context.Context,
	req *gcs.MoveObjectRequest) (*gcs.Object, error) {
	return b.wrapped.MoveObject(ctx, req)
}

// DeleteFolder passes through.
func (b *instrumentedBucket) DeleteFolder(ctx context.Context, folderName string) error {
	return b.wrapped.DeleteFolder(ctx, folderName)
}

// GetFolder passes through.
func (b *instrumentedBucket) GetFolder(ctx context.Context, req *gcs.GetFolderRequest) (*gcs.Folder, error) {
	return b.wrapped.GetFolder(ctx, req)
}

// RenameFolder passes through.
func (b *instrumentedBucket) RenameFolder(
	ctx context.Context,
	folderName string,
	destinationFolderId string) (*gcs.Folder, error) {
	return b.wrapped.RenameFolder(ctx, folderName, destinationFolderId)
}

// CreateFolder passes through.
func (b *instrumentedBucket) CreateFolder(ctx context.Context, folderName string) (*gcs.Folder, error) {
	return b.wrapped.CreateFolder(ctx, folderName)
}

// GCSName passes through.
func (b *instrumentedBucket) GCSName(object *gcs.MinObject) string {
	return b.wrapped.GCSName(object)
}

// recordEvent sends a PerfEvent to the events channel if it is set.
// The send is non-blocking: if the channel is full the event is dropped rather
// than blocking I/O goroutines.
func (b *instrumentedBucket) recordEvent(e benchmark.PerfEvent) {
	if b.events == nil {
		return
	}
	select {
	case b.events <- e:
	default:
		// Channel full — drop to avoid blocking hot-path goroutines.
	}
}

////////////////////////////////////////////////////////////////////////
// instrumentedReader
////////////////////////////////////////////////////////////////////////

// instrumentedReader wraps gcs.StorageReader to capture TTFB and total latency.
type instrumentedReader struct {
	gcs.StorageReader

	bucket     *instrumentedBucket
	start      time.Time
	ttfbOnce   bool
	ttfb       time.Duration
	objectSize int64
	bytesRead  int64
}

// Read records the TTFB on the first successful read, then accumulates bytes.
func (r *instrumentedReader) Read(p []byte) (n int, err error) {
	n, err = r.StorageReader.Read(p)
	if n > 0 {
		if !r.ttfbOnce {
			r.ttfb = time.Since(r.start)
			r.ttfbOnce = true
			r.bucket.hists.RecordTTFB(r.ttfb.Microseconds())
		}
		r.bytesRead += int64(n)
	}
	return
}

// Close records the total latency and flushes counters to the bucket.
func (r *instrumentedReader) Close() error {
	err := r.StorageReader.Close()
	elapsed := time.Since(r.start)

	r.bucket.totalOps.Add(1)
	r.bucket.totalBytes.Add(r.bytesRead)
	r.bucket.hists.RecordTotal(elapsed.Microseconds())
	r.bucket.recordEvent(benchmark.PerfEvent{
		Op:               benchmark.OpRead,
		TTFB:             r.ttfb,
		TotalLatency:     elapsed,
		BytesTransferred: r.bytesRead,
		ObjectSize:       r.objectSize,
		Err:              err,
	})
	if err != nil {
		r.bucket.totalErrs.Add(1)
	}
	return err
}

// ReadHandle delegates to the underlying reader.
func (r *instrumentedReader) ReadHandle() storagev2.ReadHandle {
	return r.StorageReader.ReadHandle()
}

// Compile-time interface check.
var _ io.ReadCloser = (*instrumentedReader)(nil)
