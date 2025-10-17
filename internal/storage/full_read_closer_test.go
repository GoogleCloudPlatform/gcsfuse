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
	"bytes"
	"io"
	"testing"

	storagev2 "cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
)

// twoBytesStorageReader reads at most 2 bytes from the buffer in one go.
type twoBytesStorageReader struct {
	buf *bytes.Buffer
}

func (psr twoBytesStorageReader) ReadHandle() (rh storagev2.ReadHandle) {
	return nil
}

func (psr twoBytesStorageReader) Close() (err error) {
	return nil
}

func (psr twoBytesStorageReader) Read(b []byte) (n int, err error) {
	maxBytes := 2
	bufLen := min(len(b), maxBytes)
	temp := make([]byte, bufLen)
	n, err = psr.buf.Read(temp)
	copy(b, temp)
	return n, err
}

func TestFullReaderCloser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		bufSize      int
		data         []byte
		expectedData []byte
		expectedErr  error
	}{
		{
			name:         "large_buffer",
			data:         []byte("0123"),
			bufSize:      5,
			expectedData: []byte("0123"),
			expectedErr:  io.EOF,
		},
		{
			name:         "small_buffer",
			data:         []byte("0123"),
			bufSize:      2,
			expectedData: []byte("01"),
			expectedErr:  nil,
		},
		{
			name:         "equal_buffer",
			data:         []byte("0123"),
			bufSize:      4,
			expectedData: []byte("0123"),
			expectedErr:  nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			storageReader := twoBytesStorageReader{
				buf: new(bytes.Buffer),
			}
			storageReader.buf.Write(tc.data)
			fullReadCloser := newGCSFullReadCloser(storageReader)
			buffer := make([]byte, tc.bufSize)

			n, err := fullReadCloser.Read(buffer)

			assert.Equal(t, len(tc.expectedData), n)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedData[:n], buffer[:n])
		})
	}
}
