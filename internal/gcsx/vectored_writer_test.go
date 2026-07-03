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

package gcsx

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVectoredWriter_Write(t *testing.T) {
	tests := []struct {
		name         string
		buffers      [][]byte
		input        []byte
		expectedN    int
		expectedErr  error
		expectedBufs [][]byte
	}{
		{
			name: "Write to single buffer fully",
			buffers: [][]byte{
				make([]byte, 5),
			},
			input:        []byte("hello"),
			expectedN:    5,
			expectedErr:  nil,
			expectedBufs: [][]byte{[]byte("hello")},
		},
		{
			name: "Write to multiple buffers fully",
			buffers: [][]byte{
				make([]byte, 3),
				make([]byte, 2),
			},
			input:        []byte("hello"),
			expectedN:    5,
			expectedErr:  nil,
			expectedBufs: [][]byte{[]byte("hel"), []byte("lo")},
		},
		{
			name: "Write with remaining space",
			buffers: [][]byte{
				make([]byte, 4),
				make([]byte, 4),
			},
			input:        []byte("hello"),
			expectedN:    5,
			expectedErr:  nil,
			expectedBufs: [][]byte{[]byte("hell"), []byte("o\x00\x00\x00")},
		},
		{
			name: "Write more than capacity",
			buffers: [][]byte{
				make([]byte, 2),
				make([]byte, 2),
			},
			input:        []byte("hello"),
			expectedN:    4,
			expectedErr:  io.ErrShortWrite,
			expectedBufs: [][]byte{[]byte("he"), []byte("ll")},
		},
		{
			name:         "Write to empty buffers list",
			buffers:      [][]byte{},
			input:        []byte("hello"),
			expectedN:    0,
			expectedErr:  io.ErrShortWrite,
			expectedBufs: [][]byte{},
		},
		{
			name: "Write empty data",
			buffers: [][]byte{
				make([]byte, 5),
			},
			input:        []byte(""),
			expectedN:    0,
			expectedErr:  nil,
			expectedBufs: [][]byte{make([]byte, 5)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := NewVectoredWriter(tc.buffers)

			n, err := w.Write(tc.input)

			assert.Equal(t, tc.expectedN, n)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedBufs, tc.buffers)
		})
	}
}

func TestGetVectoredBuffers(t *testing.T) {
	tests := []struct {
		name         string
		req          *ReadRequest
		limit        int64
		expectedBufs [][]byte
		expectedSize int64
	}{
		{
			name: "Single buffer, no limit, size matches capacity",
			req: &ReadRequest{
				Buffer: make([]byte, 10),
				Size:   10,
			},
			limit:        0,
			expectedBufs: [][]byte{make([]byte, 10)},
			expectedSize: 10,
		},
		{
			name: "Single buffer, size less than capacity",
			req: &ReadRequest{
				Buffer: make([]byte, 10),
				Size:   5,
			},
			limit:        0,
			expectedBufs: [][]byte{make([]byte, 5)},
			expectedSize: 5,
		},
		{
			name: "Single buffer, size 0 (defaults to capacity)",
			req: &ReadRequest{
				Buffer: make([]byte, 10),
				Size:   0,
			},
			limit:        0,
			expectedBufs: [][]byte{make([]byte, 10)},
			expectedSize: 10,
		},
		{
			name: "Single buffer, size exceeds capacity (capped to capacity)",
			req: &ReadRequest{
				Buffer: make([]byte, 10),
				Size:   15,
			},
			limit:        0,
			expectedBufs: [][]byte{make([]byte, 10)},
			expectedSize: 10,
		},
		{
			name: "Single buffer, limit applies",
			req: &ReadRequest{
				Buffer: make([]byte, 10),
				Size:   8,
			},
			limit:        5,
			expectedBufs: [][]byte{make([]byte, 5)},
			expectedSize: 5,
		},
		{
			name: "Multi buffer, no limit, size matches capacity",
			req: &ReadRequest{
				Buffers: [][]byte{
					make([]byte, 3),
					make([]byte, 4),
				},
				Size: 7,
			},
			limit: 0,
			expectedBufs: [][]byte{
				make([]byte, 3),
				make([]byte, 4),
			},
			expectedSize: 7,
		},
		{
			name: "Multi buffer, size less than capacity (returns all buffers, caller limits write)",
			req: &ReadRequest{
				Buffers: [][]byte{
					make([]byte, 3),
					make([]byte, 4),
				},
				Size: 5,
			},
			limit: 0,
			expectedBufs: [][]byte{
				make([]byte, 3),
				make([]byte, 4),
			},
			expectedSize: 5,
		},
		{
			name: "Multi buffer, limit applies",
			req: &ReadRequest{
				Buffers: [][]byte{
					make([]byte, 3),
					make([]byte, 4),
				},
				Size: 7,
			},
			limit: 5,
			expectedBufs: [][]byte{
				make([]byte, 3),
				make([]byte, 4),
			},
			expectedSize: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bufs, size := GetVectoredBuffers(tc.req, tc.limit)

			assert.Equal(t, tc.expectedSize, size)
			assert.Equal(t, tc.expectedBufs, bufs)
		})
	}
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func TestVectoredWriter_ReadFrom(t *testing.T) {
	testErr := errors.New("read error")

	tests := []struct {
		name         string
		buffers      [][]byte
		reader       io.Reader
		expectedN    int64
		expectedErr  error
		expectedBufs [][]byte
	}{
		{
			name: "ReadFrom single buffer fully",
			buffers: [][]byte{
				make([]byte, 5),
			},
			reader:       bytes.NewReader([]byte("hello")),
			expectedN:    5,
			expectedErr:  nil,
			expectedBufs: [][]byte{[]byte("hello")},
		},
		{
			name: "ReadFrom multiple buffers fully",
			buffers: [][]byte{
				make([]byte, 3),
				make([]byte, 2),
			},
			reader:       bytes.NewReader([]byte("hello")),
			expectedN:    5,
			expectedErr:  nil,
			expectedBufs: [][]byte{[]byte("hel"), []byte("lo")},
		},
		{
			name: "ReadFrom with remaining space",
			buffers: [][]byte{
				make([]byte, 4),
				make([]byte, 4),
			},
			reader:       bytes.NewReader([]byte("hello")),
			expectedN:    5,
			expectedErr:  nil,
			expectedBufs: [][]byte{[]byte("hell"), []byte("o\x00\x00\x00")},
		},
		{
			name: "ReadFrom more than capacity",
			buffers: [][]byte{
				make([]byte, 2),
				make([]byte, 2),
			},
			reader:       bytes.NewReader([]byte("hello")),
			expectedN:    4,
			expectedErr:  nil,
			expectedBufs: [][]byte{[]byte("he"), []byte("ll")},
		},
		{
			name:         "ReadFrom to empty buffers list",
			buffers:      [][]byte{},
			reader:       bytes.NewReader([]byte("hello")),
			expectedN:    0,
			expectedErr:  nil,
			expectedBufs: [][]byte{},
		},
		{
			name: "ReadFrom empty reader",
			buffers: [][]byte{
				make([]byte, 5),
			},
			reader:       bytes.NewReader([]byte("")),
			expectedN:    0,
			expectedErr:  nil,
			expectedBufs: [][]byte{make([]byte, 5)},
		},
		{
			name: "ReadFrom reader error",
			buffers: [][]byte{
				make([]byte, 5),
			},
			reader:       &errorReader{err: testErr},
			expectedN:    0,
			expectedErr:  testErr,
			expectedBufs: [][]byte{make([]byte, 5)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := NewVectoredWriter(tc.buffers)

			n, err := w.ReadFrom(tc.reader)

			assert.Equal(t, tc.expectedN, n)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedBufs, tc.buffers)
		})
	}
}
