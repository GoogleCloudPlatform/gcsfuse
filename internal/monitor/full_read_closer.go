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
// if size of the buffer is more than the response, then the entire response is returned,
// or if size of the buffer is less than or equal to the response, then the entire buffer is filled,
// or an error is encountered.
type gcsFullReadCloser struct {
	wrapped gcs.StorageReader
}

func newGCSFullReadCloser(reader gcs.StorageReader) gcs.StorageReader {
	return gcsFullReadCloser{wrapped: reader}
}

// Read reads exactly len(buf) bytes from the wrapped StorageReader into buf.
// It returns the number of bytes copied and an error if fewer bytes were read.
// The error is EOF only if no bytes were read.
// If an EOF happens after reading some but not all the bytes, Read returns ErrUnexpectedEOF.
// On return, n == len(buf) if and only if err == nil.
// If StorageReader returns an error having read at least len(buf) bytes, the error is dropped.
func (frc gcsFullReadCloser) Read(buf []byte) (n int, err error) {
	return io.ReadFull(frc.wrapped, buf)
}

func (frc gcsFullReadCloser) ReadHandle() (rh storagev2.ReadHandle) {
	return frc.wrapped.ReadHandle()
}

func (frc gcsFullReadCloser) Close() (err error) {
	return frc.wrapped.Close()
}
