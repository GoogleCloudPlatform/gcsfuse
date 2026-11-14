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
	"io"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
)

// gcsFullReadCloser wraps storage.Reader (initially conforms to io.ReadCloser)
// and forces Read to call io.ReadFull instead.
type gcsFullReadCloser struct {
	wrapped gcs.StorageReader
}

func newGCSFullReadCloser(StorageReader gcs.StorageReader) gcsFullReadCloser {
	return gcsFullReadCloser{
		wrapped: StorageReader,
	}
}

// Re-define the behavior of Read function to return:
// 1. the number of bytes copied and an EOF if response size < buffer size
// 2. n == len(buf) if and only if err == nil.
func (frc gcsFullReadCloser) Read(buf []byte) (n int, err error) {
	n, err = io.ReadFull(frc.wrapped, buf)
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	return n, err
}

// ReadVectored reads up to len(buf) bytes from a gRPC-backed *storagev2.Reader
// by referencing the underlying gRPC mem.BufferSlice directly without copying
// payload bytes into buf.
func (frc gcsFullReadCloser) ReadVectored(buf []byte) (data [][]byte, n int, callback func(), err error) {
	if len(buf) == 0 {
		return nil, 0, nil, nil
	}
	if sr, ok := frc.wrapped.(*storagev2.Reader); ok {
		return sr.ReadVectored(buf)
	}
	n, err = frc.Read(buf)
	return nil, n, nil, err
}

func (frc gcsFullReadCloser) ReadHandle() (rh storagev2.ReadHandle) {
	return frc.wrapped.ReadHandle()
}

func (frc gcsFullReadCloser) Close() (err error) {
	return frc.wrapped.Close()
}
