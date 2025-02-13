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
	"bytes"
	"io"
	"testing"

	storagev2 "cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
)

type dummyStorageReader struct {
	buf      *bytes.Buffer
	maxBytes int
}

func (frc dummyStorageReader) ReadHandle() (rh storagev2.ReadHandle) {
	return nil
}

func (frc dummyStorageReader) Close() (err error) {
	return nil
}

func (frc dummyStorageReader) Read(b []byte) (n int, err error) {
	bufLen := min(len(b), frc.maxBytes)
	temp := make([]byte, bufLen)
	n, err = frc.buf.Read(temp)
	copy(b, temp)
	return n, err
}

func TestFullReaderCloser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                 string
		bufSize              int
		data                 []byte
		maxReadBytes         int
		expectedData         []byte
		expectedErr          error
		expectedNumBytesRead int
	}{
		{
			name:                 "large_buffer",
			data:                 []byte("0123"),
			bufSize:              5,
			maxReadBytes:         10,
			expectedData:         []byte("0123"),
			expectedErr:          io.ErrUnexpectedEOF,
			expectedNumBytesRead: 4,
		},
		{
			name:                 "small_buffer",
			data:                 []byte("0123"),
			bufSize:              2,
			maxReadBytes:         10,
			expectedData:         []byte("01"),
			expectedErr:          nil,
			expectedNumBytesRead: 2,
		},
		{
			name:                 "equal_buffer",
			data:                 []byte("0123"),
			bufSize:              4,
			maxReadBytes:         10,
			expectedData:         []byte("0123"),
			expectedErr:          nil,
			expectedNumBytesRead: 4,
		},
		{
			name:                 "partial_read_full_data_returned",
			data:                 []byte("0123"),
			bufSize:              10,
			maxReadBytes:         2,
			expectedData:         []byte("0123"),
			expectedErr:          io.ErrUnexpectedEOF,
			expectedNumBytesRead: 4,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			storageReader := dummyStorageReader{
				maxBytes: tc.maxReadBytes,
				buf:      new(bytes.Buffer),
			}
			storageReader.buf.Write(tc.data)
			fullReadCloser := newGCSFullReadCloser(storageReader)
			buffer := make([]byte, tc.bufSize)

			n, err := fullReadCloser.Read(buffer)

			assert.Equal(t, tc.expectedNumBytesRead, n)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedData[:n], buffer[:n])
		})
	}
}
