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

package monitor

import (
	"io"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
)

// gcsFullReadCloser wraps a gcs.StorageReader and ensures that the Read call returns:
// 1. entire response, if buffer size > response size
// 2. entire buffer is filled, if buffer size <= response size
// 3. error
type gcsFullReadCloser struct {
	wrapped gcs.StorageReader
}

func newGCSFullReadCloser(reader gcs.StorageReader) gcs.StorageReader {
	return gcsFullReadCloser{wrapped: reader}
}

// Read reads exactly len(buf) bytes from the wrapped StorageReader into buf.
// 1. the number of bytes copied and an ErrUnexpectedEOF if response size < buffer size
// 2. EOF only if no bytes were read.
// 3. n == len(buf) if and only if err == nil.
func (frc gcsFullReadCloser) Read(buf []byte) (n int, err error) {
	return io.ReadFull(frc.wrapped, buf)
}

func (frc gcsFullReadCloser) ReadHandle() (rh storagev2.ReadHandle) {
	return frc.wrapped.ReadHandle()
}

func (frc gcsFullReadCloser) Close() (err error) {
	return frc.wrapped.Close()
}
