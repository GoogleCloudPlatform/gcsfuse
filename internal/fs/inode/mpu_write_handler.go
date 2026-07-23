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

package inode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/util"
)

var ErrOutOfOrderWrite = errors.New("outOfOrder write detected")

// MPUWriteHandler defines operations for gRPC parallel upload streaming (MPU).
type MPUWriteHandler interface {
	Write(ctx context.Context, data []byte, offset int64) (n int64, err error)
	Truncate(ctx context.Context, size int64) (fallback bool, err error)
	Sync(ctx context.Context) error
	TotalSize() int64
	Close() error
	Abort(ctx context.Context) error
}

type mpuWriteHandlerImpl struct {
	writer        gcs.ParallelUploadWriter
	totalSize     int64
	truncatedSize int64
}

func NewMPUWriteHandler(writer gcs.ParallelUploadWriter) MPUWriteHandler {
	return &mpuWriteHandlerImpl{
		writer:        writer,
		totalSize:     0,
		truncatedSize: -1,
	}
}

func (h *mpuWriteHandlerImpl) TotalSize() int64 {
	if h.truncatedSize != -1 {
		return max(h.totalSize, h.truncatedSize)
	}
	return h.totalSize
}

func (h *mpuWriteHandlerImpl) Write(ctx context.Context, data []byte, offset int64) (n int64, err error) {
	if h.truncatedSize != -1 && offset == h.truncatedSize {
		if err := h.padZeroBytesUpTo(ctx, h.truncatedSize); err != nil {
			return 0, err
		}
		h.truncatedSize = -1
	}

	if offset != h.totalSize {
		return 0, ErrOutOfOrderWrite
	}

	n, err = io.Copy(h.writer, bytes.NewReader(data))
	if err != nil {
		return 0, fmt.Errorf("mpuWriter.Write(): %w", err)
	}

	h.totalSize += n
	if h.truncatedSize != -1 && h.totalSize >= h.truncatedSize {
		h.truncatedSize = -1
	}
	return n, nil
}

func (h *mpuWriteHandlerImpl) Truncate(ctx context.Context, size int64) (fallback bool, err error) {
	effectiveSize := h.TotalSize()
	if size == effectiveSize {
		return false, nil
	}

	if size < h.totalSize {
		// Shrink: finalize existing stream and signal fallback to temp file.
		if err := h.writer.Close(); err != nil {
			return true, fmt.Errorf("mpuWriter.Close() on shrink: %w", err)
		}
		return true, nil
	}

	// Extend: store lazy truncation offset.
	h.truncatedSize = size
	return false, nil
}

func (h *mpuWriteHandlerImpl) Sync(ctx context.Context) error {
	if h.truncatedSize != -1 && h.truncatedSize > h.totalSize {
		if err := h.padZeroBytesUpTo(ctx, h.truncatedSize); err != nil {
			return err
		}
		h.truncatedSize = -1
	}
	return nil
}

func (h *mpuWriteHandlerImpl) Close() error {
	return h.writer.Close()
}

func (h *mpuWriteHandlerImpl) Abort(ctx context.Context) error {
	return h.writer.Abort(ctx)
}

func (h *mpuWriteHandlerImpl) padZeroBytesUpTo(ctx context.Context, targetOffset int64) error {
	paddingNeeded := targetOffset - h.totalSize
	if paddingNeeded <= 0 {
		return nil
	}

	zeroBuf := make([]byte, min(paddingNeeded, 1*util.MiB))
	for paddingNeeded > 0 {
		toWrite := min(paddingNeeded, int64(len(zeroBuf)))
		n, err := io.Copy(h.writer, bytes.NewReader(zeroBuf[:toWrite]))
		if err != nil {
			return fmt.Errorf("mpuWriter zero-padding failed: %w", err)
		}
		h.totalSize += n
		paddingNeeded -= n
	}
	return nil
}
