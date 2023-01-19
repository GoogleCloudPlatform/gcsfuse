// Copyright 2015 Google Inc. All Rights Reserved.
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
	"fmt"
	"io"
	"log"

	"github.com/googlecloudplatform/gcsfuse/internal/monitor/tags"
	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"github.com/jacobsa/gcloud/gcs"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"golang.org/x/net/context"
)

var (
	// When a first read call is made by the user, we either fetch entire file or x number of bytes from GCS based on the request.
	// Now depending on the pagesize multiple read calls will be issued by user to read the entire file. These
	// requests will be served from the downloaded data.
	// This metric captures only the requests made to GCS, not the subsequent page calls.
	gcsReadCount = stats.Int64("gcs/read_count",
		"Specifies the count of gcs reads made along with type",
		stats.UnitDimensionless)
	downloadBytesCount = stats.Int64("gcs/download_bytes_count",
		"Cumulative number of bytes downloaded from GCS along with read type",
		stats.UnitBytes)
)

// MB is 1 Megabyte. (Silly comment to make the lint warning go away)
const MB = 1 << 20

// Min read size in bytes for random reads.
// We will not send a request to GCS for less than this many bytes (unless the
// end of the object comes first).
const minReadSize = MB

// Max read size in bytes for random reads.
// If the average read size (between seeks) is below this number, reads will
// optimised for random access.
// We will skip forwards in a GCS response at most this many bytes.
// About 6 MB of data is buffered anyway, so 8 MB seems like a good round number.
const maxReadSize = 8 * MB

// Minimum number of seeks before evaluating if the read pattern is random.
const minSeeksForRandom = 2

// Constants for read types - sequential/random
const sequential = "Sequential"
const random = "Random"

// Initialize the metrics.
func init() {
	if err := view.Register(
		&view.View{
			Name:        "gcs/read_count",
			Measure:     gcsReadCount,
			Description: "Specifies the number of gcs reads made along with type- Sequential/Random",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.ReadType},
		},
		&view.View{
			Name:        "gcs/download_bytes_count",
			Measure:     downloadBytesCount,
			Description: "The cumulative number of bytes downloaded from GCS.",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.ReadType},
		},
	); err != nil {
		log.Fatalf("Failed to register the view: %v", err)
	}
}

// RandomReader is an object that knows how to read ranges within a particular
// generation of a particular GCS object. Optimised for (large) sequential reads.
//
// Not safe for concurrent access.
type RandomReader interface {
	// Panic if any internal invariants are violated.
	CheckInvariants()

	// Matches the semantics of io.ReaderAt, with the addition of context
	// support.
	ReadAt(ctx context.Context, p []byte, offset int64) (n int, err error)

	// Return the record for the object to which the reader is bound.
	Object() (o *storage.MinObject)

	// Clean up any resources associated with the reader, which must not be used
	// again.
	Destroy()
}

// NewRandomReader create a random reader for the supplied object record that
// reads using the given bucket.
func NewRandomReader(o *storage.MinObject, bucket gcs.Bucket, sequentialReadSizeMb int32) RandomReader {
	return &randomReader{
		object:               o,
		bucket:               bucket,
		start:                -1,
		limit:                -1,
		seeks:                0,
		totalReadBytes:       0,
		sequentialReadSizeMb: sequentialReadSizeMb,
	}
}

type randomReader struct {
	object *storage.MinObject
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
	start          int64
	limit          int64
	seeks          uint64
	totalReadBytes uint64

	sequentialReadSizeMb int32
}

func (rr *randomReader) CheckInvariants() {
	// INVARIANT: (reader == nil) == (cancel == nil)
	if (rr.reader == nil) != (rr.cancel == nil) {
		panic(fmt.Sprintf("Mismatch: %v vs. %v", rr.reader == nil, rr.cancel == nil))
	}

	// INVARIANT: start <= limit
	if !(rr.start <= rr.limit) {
		panic(fmt.Sprintf("Unexpected range: [%d, %d)", rr.start, rr.limit))
	}

	// INVARIANT: limit < 0 implies reader != nil
	if rr.limit < 0 && rr.reader != nil {
		panic(fmt.Sprintf("Unexpected non-nil reader with limit == %d", rr.limit))
	}
}

func (rr *randomReader) ReadAt(
	ctx context.Context,
	p []byte,
	offset int64) (n int, err error) {
	for len(p) > 0 {
		// Have we blown past the end of the object?
		if offset >= int64(rr.object.Size) {
			err = io.EOF
			return
		}

		// When the offset is AFTER the reader position, try to seek forward, within reason.
		// This happens when the kernel page cache serves some data. It's very common for
		// concurrent reads, often by only a few 128kB fuse read requests. The aim is to
		// re-use GCS connection and avoid throwing away already read data.
		// For parallel sequential reads to a single file, not throwing away the connections
		// is a 15-20x improvement in throughput: 150-200 MB/s instead of 10 MB/s.
		if rr.reader != nil && rr.start < offset && offset-rr.start < maxReadSize {
			bytesToSkip := int64(offset - rr.start)
			p := make([]byte, bytesToSkip)
			n, _ := rr.reader.Read(p)
			rr.start += int64(n)
		}

		// If we have an existing reader but it's positioned at the wrong place,
		// clean it up and throw it away.
		if rr.reader != nil && rr.start != offset {
			rr.reader.Close()
			rr.reader = nil
			rr.cancel = nil
			rr.seeks++
		}

		// If we don't have a reader, start a read operation.
		if rr.reader == nil {
			err = rr.startRead(ctx, offset, int64(len(p)))
			if err != nil {
				err = fmt.Errorf("startRead: %w", err)
				return
			}
		}

		// Now we have a reader positioned at the correct place. Consume as much from
		// it as possible.
		var tmp int
		tmp, err = rr.readFull(ctx, p)

		n += tmp
		p = p[tmp:]
		rr.start += int64(tmp)
		offset += int64(tmp)
		rr.totalReadBytes += uint64(tmp)

		// Sanity check.
		if rr.start > rr.limit {
			err = fmt.Errorf("Reader returned %d too many bytes", rr.start-rr.limit)

			// Don't attempt to reuse the reader when it's behaving wackily.
			rr.reader.Close()
			rr.reader = nil
			rr.cancel = nil
			rr.start = -1
			rr.limit = -1

			return
		}

		// Are we finished with this reader now?
		if rr.start == rr.limit {
			rr.reader.Close()
			rr.reader = nil
			rr.cancel = nil
		}

		// Handle errors.
		switch {
		case err == io.EOF || err == io.ErrUnexpectedEOF:
			// For a non-empty buffer, ReadFull returns EOF or ErrUnexpectedEOF only
			// if the reader peters out early. That's fine, but it means we should
			// have hit the limit above.
			if rr.reader != nil {
				err = fmt.Errorf("Reader returned %d too few bytes", rr.limit-rr.start)
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

func (rr *randomReader) Object() (o *storage.MinObject) {
	o = rr.object
	return
}

func (rr *randomReader) Destroy() {
	// Close out the reader, if we have one.
	if rr.reader != nil {
		rr.reader.Close()
		rr.reader = nil
		rr.cancel = nil
	}
}

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

// Ensure that rr.reader is set up for a range for which [start, start+size) is
// a prefix. Irrespective of the size requested, we try to fetch more data
// from GCS defined by sequentialReadSizeMb flag to serve future read requests.
func (rr *randomReader) startRead(
	ctx context.Context,
	start int64,
	size int64) (err error) {
	// Make sure start and size are legal.
	if start < 0 || uint64(start) > rr.object.Size || size < 0 {
		err = fmt.Errorf(
			"Range [%d, %d) is illegal for %d-byte object",
			start,
			start+size,
			rr.object.Size)
		return
	}

	// GCS requests are expensive. Prefer to issue read requests defined by
	// sequentialReadSizeMb flag. Sequential reads will simply sip from the fire house
	// with each call to ReadAt. In practice, GCS will fill the TCP buffers
	// with about 6 MB of data. Requests from outside GCP will be charged
	// about 6MB of egress data, even if less data is read. Inside GCP
	// regions, GCS egress is free. This logic should limit the number of
	// GCS read requests, which are not free.

	// But if we notice random read patterns after a minimum number of seeks,
	// optimise for random reads. Random reads will read data in chunks of
	// (average read size in bytes rounded up to the next MB).
	end := int64(rr.object.Size)
	readType := sequential
	if rr.seeks >= minSeeksForRandom {
		readType = random
		averageReadBytes := rr.totalReadBytes / rr.seeks
		if averageReadBytes < maxReadSize {
			randomReadSize := int64(((averageReadBytes / MB) + 1) * MB)
			if randomReadSize < minReadSize {
				randomReadSize = minReadSize
			}
			if randomReadSize > maxReadSize {
				randomReadSize = maxReadSize
			}
			end = start + randomReadSize
		}
	}
	if end > int64(rr.object.Size) {
		end = int64(rr.object.Size)
	}

	// To avoid overloading GCS and to have reasonable latencies, we will only
	// fetch data of max size defined by sequentialReadSizeMb.
	maxSizeToReadFromGCS := int64(rr.sequentialReadSizeMb * MB)
	if end-start > maxSizeToReadFromGCS {
		end = start + maxSizeToReadFromGCS
	}

	// Begin the read.
	ctx, cancel := context.WithCancel(context.Background())
	rc, err := rr.bucket.NewReader(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       rr.object.Name,
			Generation: rr.object.Generation,
			Range: &gcs.ByteRange{
				Start: uint64(start),
				Limit: uint64(end),
			},
		})

	if err != nil {
		err = fmt.Errorf("NewReader: %w", err)
		return
	}

	rr.reader = rc
	rr.cancel = cancel
	rr.start = start
	rr.limit = end

	requestedDataSize := end - start
	captureMetrics(ctx, readType, requestedDataSize)

	return
}

func captureMetrics(ctx context.Context, readType string, requestedDataSize int64) {
	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.ReadType, readType),
		},
		gcsReadCount.M(1),
	); err != nil {
		// Error in recording gcsReadCount.
		log.Fatalf("Cannot record gcsReadCount %v", err)
	}

	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.ReadType, readType),
		},
		downloadBytesCount.M(requestedDataSize),
	); err != nil {
		// Error in recording gcsReadCount.
		log.Fatalf("Cannot record downloadBytesCount %v", err)
	}
}
