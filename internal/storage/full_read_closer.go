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
	"github.com/vipnydav/gcsfuse/v3/internal/storage/gcs"
)

// gcsFullReadCloser wraps a gcs.StorageReader and ensures that the Read call reads the entire response up to the buffer size even if the wrapped read returns data in smaller chunks.
type gcsFullReadCloser struct {
	wrapped gcs.StorageReader
}

func newGCSFullReadCloser(reader gcs.StorageReader) gcs.StorageReader {
	return gcsFullReadCloser{wrapped: reader}
}

// Read reads exactly len(buf) bytes from the wrapped StorageReader into buf.
// 1. the number of bytes copied and an EOF if response size < buffer size
// 2. n == len(buf) if and only if err == nil.
func (frc gcsFullReadCloser) Read(buf []byte) (n int, err error) {
	n, err = io.ReadFull(frc.wrapped, buf)
	if err == io.ErrUnexpectedEOF {
		// if an EOF is encountered before reading the full length of the buffer,
		// ReadFull returns an ErrUnexpectedEOF error. This needs to be convered
		// to EOF in order to have a consistent behavior (error) with and without gcsFullReadCloser.
		err = io.EOF
	}
	return n, err
}

func (frc gcsFullReadCloser) ReadHandle() (rh storagev2.ReadHandle) {
	return frc.wrapped.ReadHandle()
}

func (frc gcsFullReadCloser) Close() (err error) {
	return frc.wrapped.Close()
}
