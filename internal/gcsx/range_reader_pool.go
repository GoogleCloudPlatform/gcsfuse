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
	"context"
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// RangeReaderPool manages a pool of RangeReaders for a specific object.
type RangeReaderPool struct {
	bucket gcs.Bucket
	object *gcs.MinObject

	mu sync.Mutex // Guards the fields below.
	// readers holds the inactive GCS readers.
	readers []gcs.StorageReader
	// refCount tracks the number of active users of the pool.
	refCount int
}

// NewRangeReaderPool creates a new RangeReaderPool.
func NewRangeReaderPool(bucket gcs.Bucket, object *gcs.MinObject) *RangeReaderPool {
	return &RangeReaderPool{
		bucket:  bucket,
		object:  object,
		readers: make([]gcs.StorageReader, 0),
	}
}

// IncrementRefCount increments the reference count of the pool.
func (p *RangeReaderPool) IncrementRefCount() {
	// p.mu.Lock()
	// defer p.mu.Unlock()
	// p.refCount++
}

// DecrementRefCount decrements the reference count and cleans up if it reaches zero.
func (p *RangeReaderPool) DecrementRefCount() {
	// p.mu.Lock()
	// defer p.mu.Unlock()
	// p.refCount--
	// if p.refCount <= 0 {
	// 	for _, r := range p.readers {
	// 		_ = r.Close() // Ignore error on close.
	// 	}
	// 	p.readers = nil
	// }
}

// Checkout returns an available RangeReader or creates a new one.
func (p *RangeReaderPool) Checkout(ctx context.Context, start, limit uint64) (gcs.StorageReader, error) {
	// Note: Readers from the pool cannot be reused as they are tied to a specific
	// range. Therefore, we always create a new reader for the requested range.

	// Create a new GCS reader if none are available.
	reader, err := p.bucket.NewReaderWithReadHandle(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       p.object.Name,
			Generation: p.object.Generation,
			Range: &gcs.ByteRange{
				Start: start,
				Limit: limit,
			},
			ReadCompressed: p.object.HasContentEncodingGzip(),
		})
	if err != nil {
		return nil, err
	}
	return reader, nil
}

// Checkin returns a RangeReader to the pool.
func (p *RangeReaderPool) Checkin(r gcs.StorageReader) {
	// Since the reader is for a specific range and likely consumed,
	// we close it instead of returning it to the pool for reuse.
	_ = r.Close()
}
