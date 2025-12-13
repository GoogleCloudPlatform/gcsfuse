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
	"fmt"
	"io"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// PooledRangeReader implements the Reader interface using a RangeReaderPool.
type PooledRangeReader struct {
	pool   *RangeReaderPool
	object *gcs.MinObject
}

// NewPooledRangeReader creates a new PooledRangeReader.
func NewPooledRangeReader(pool *RangeReaderPool, object *gcs.MinObject) *PooledRangeReader {
	return &PooledRangeReader{
		pool:   pool,
		object: object,
	}
}

// CheckInvariants performs internal consistency checks on the reader state.
func (pr *PooledRangeReader) CheckInvariants() {
	if pr.object == nil {
		panic("PooledRangeReader: object is nil")
	}
	if pr.pool == nil {
		panic("PooledRangeReader: pool is nil")
	}
}

// ReadAt reads data from the object using a RangeReader from the pool.
func (pr *PooledRangeReader) ReadAt(ctx context.Context, req *ReadRequest) (ReadResponse, error) {
	var resp ReadResponse

	if req.Offset >= int64(pr.object.Size) {
		return resp, io.EOF
	}

	endOffset := req.Offset + int64(len(req.Buffer))
	if endOffset > int64(pr.object.Size) {
		endOffset = int64(pr.object.Size)
	}

	rr, err := pr.pool.Checkout(ctx, uint64(req.Offset), uint64(endOffset))
	if err != nil {
		return resp, fmt.Errorf("failed to checkout range reader: %w", err)
	}
	defer pr.pool.Checkin(rr)

	n, err := io.ReadFull(rr, req.Buffer[:endOffset-req.Offset])
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		// The GCS reader returns io.ErrUnexpectedEOF if the object size is smaller
		// than the buffer. We should treat this as a successful read of a smaller
		// number of bytes, and not an error.
		err = nil
	}
	resp.Size = n
	return resp, err
}

// Destroy releases resources.
func (pr *PooledRangeReader) Destroy() {
}

// Object returns the object metadata.
func (pr *PooledRangeReader) Object() *gcs.MinObject {
	return pr.object
}
