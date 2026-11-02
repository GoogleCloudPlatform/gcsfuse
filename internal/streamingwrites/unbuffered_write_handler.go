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
)

// errNotImplemented is returned by the unbufferedWriteHandlerImpl methods that
// are not implemented yet.
// TODO(b/576158185): Remove once the unbuffered write handler is implemented.
var errNotImplemented = errors.New("unbuffered write handler: not implemented")

// unbufferedWriteHandlerImpl is the WriteHandler for parallel uploads. It will
// pass writes straight to a Go SDK writer, which uploads the object in parts
// concurrently, so GCSFuse keeps no write buffers of its own. Only sequential
// writes will be supported. Other writes will return ErrOutOfOrderWrite, so
// that the caller falls back to staged writes.
//
// All methods assume the caller holds the FileInode lock, so no internal
// synchronization is needed.
//
// TODO(b/576158185): Implement Write, Sync, Flush, Truncate, Destroy and
// Unlink. Nothing creates this handler yet.
type unbufferedWriteHandlerImpl struct {
	// totalSize is the number of bytes handed to the writer so far, including
	// any zero padding already written.
	totalSize int64
	// truncatedSize is the size requested by a pending forward truncate, or -1
	// if there is none. The gap up to it is zero-padded lazily.
	truncatedSize int64
	// mtime is the file's mtime. It starts at the handler's creation time and is
	// updated by the kernel through SetInodeAttributes.
	mtime time.Time
}

var _ WriteHandler = (*unbufferedWriteHandlerImpl)(nil)

// newUnbufferedWriteHandler creates an unbufferedWriteHandlerImpl for the
// object described by req. The handler is only used for new files and empty
// objects, so totalSize starts at 0. A later write at any other offset is out
// of order and falls back to staged writes, so an existing object's data is
// never dropped.
func newUnbufferedWriteHandler(req *CreateWriteHandlerRequest) *unbufferedWriteHandlerImpl {
	return &unbufferedWriteHandlerImpl{
		totalSize:     0,
		truncatedSize: -1,
		mtime:         time.Now(),
	}
}

func (wh *unbufferedWriteHandlerImpl) Write(ctx context.Context, data []byte, offset int64) error {
	return errNotImplemented
}

func (wh *unbufferedWriteHandlerImpl) Sync(ctx context.Context) (*gcs.MinObject, error) {
	return nil, errNotImplemented
}

func (wh *unbufferedWriteHandlerImpl) Flush(ctx context.Context) (*gcs.MinObject, error) {
	return nil, errNotImplemented
}

func (wh *unbufferedWriteHandlerImpl) SetMtime(mtime time.Time) {
	wh.mtime = mtime
}

func (wh *unbufferedWriteHandlerImpl) Truncate(size int64) error {
	return errNotImplemented
}

func (wh *unbufferedWriteHandlerImpl) WriteFileInfo() WriteFileInfo {
	return WriteFileInfo{
		TotalSize: max(wh.totalSize, wh.truncatedSize),
		Mtime:     wh.mtime,
	}
}

func (wh *unbufferedWriteHandlerImpl) Destroy() error {
	return errNotImplemented
}

func (wh *unbufferedWriteHandlerImpl) Unlink() {
	// TODO(b/576158185): Abort the upload and discard later writes.
}
