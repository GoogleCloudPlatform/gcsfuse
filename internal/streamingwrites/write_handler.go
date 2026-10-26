// Copyright 2026 Google LLC
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

package streamingwrites

import (
	"context"
	"errors"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"golang.org/x/sync/semaphore"
)

// Note: All the write operations take inode lock in fs.go, hence we don't need any locks here
// as we will get write operations serially.

type WriteHandler interface {
	// Write writes the given data to the underlying write handler.
	Write(ctx context.Context, data []byte, offset int64) (err error)

	// Sync synchronizes pending data with GCS.
	// Sync returns:
	// 1. un-finalized object created on GCS for zonal/appendable buckets.
	// 2. nil object for non-zonal/parallel-upload buckets.
	Sync(ctx context.Context) (*gcs.MinObject, error)

	// Flush finalizes the upload.
	Flush(ctx context.Context) (*gcs.MinObject, error)

	// SetMtime stores the mtime with the writeHandler.
	SetMtime(mtime time.Time)

	// Truncate allows truncating the file to a larger size.
	Truncate(size int64) error

	// WriteFileInfo returns the file info i.e, how much data has been written/buffered
	// so far and the mtime.
	WriteFileInfo() WriteFileInfo

	// Destroy cleans up resources held by the write handler when write handles are closed.
	Destroy() error

	// Unlink cancels/aborts the ongoing upload and frees up any local buffers.
	Unlink()
}

// WriteFileInfo is used as part of serving fileInode attributes (GetInodeAttributes call).
type WriteFileInfo struct {
	TotalSize int64
	Mtime     time.Time
}

var ErrOutOfOrderWrite = errors.New("outOfOrder write detected")

type CreateWriteHandlerRequest struct {
	Object                   *gcs.Object
	ObjectName               string
	Bucket                   gcs.Bucket
	BlockSize                int64
	MaxBlocksPerFile         int64
	GlobalMaxBlocksSem       *semaphore.Weighted
	ChunkRetryDeadlineSecs   int64
	ChunkTransferTimeoutSecs int64
	TraceHandle              tracing.TraceHandle
}

// NewWriteHandler creates the appropriate WriteHandler implementation (buffered or unbuffered).
func NewWriteHandler(req *CreateWriteHandlerRequest) (WriteHandler, error) {
	// TODO(b/548851494): Route to newUnbufferedWriteHandler(req) when parallel upload mode is enabled.
	return newBufferedWriteHandler(req)
}
