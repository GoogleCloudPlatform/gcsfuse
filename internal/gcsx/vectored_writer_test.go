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
			expectedBufs: [][]byte{[]byte("hell"), []byte("o")},
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
			expectedBufs: [][]byte{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var maxSize int64
			for _, b := range tc.buffers {
				maxSize += int64(len(b))
			}
			var pool BufferPool
			if len(tc.buffers) > 0 {
				pool = &TestBufferPool{Buffers: tc.buffers}
			}
			w := NewVectoredWriter(pool, maxSize)

			n, err := w.Write(tc.input)

			assert.Equal(t, tc.expectedN, n)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedBufs, w.Buffers())
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
			expectedBufs: [][]byte{[]byte("hell"), []byte("o")},
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
			expectedBufs: [][]byte{},
		},
		{
			name: "ReadFrom reader error",
			buffers: [][]byte{
				make([]byte, 5),
			},
			reader:       &errorReader{err: testErr},
			expectedN:    0,
			expectedErr:  testErr,
			expectedBufs: [][]byte{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var maxSize int64
			for _, b := range tc.buffers {
				maxSize += int64(len(b))
			}
			var pool BufferPool
			if len(tc.buffers) > 0 {
				pool = &TestBufferPool{Buffers: tc.buffers}
			}
			w := NewVectoredWriter(pool, maxSize)

			n, err := w.ReadFrom(tc.reader)

			assert.Equal(t, tc.expectedN, n)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedBufs, w.Buffers())
		})
	}
}

type mockReaderAt struct {
	data []byte
	err  error
}

func (m *mockReaderAt) ReadAt(dst []byte, offset int64) (n int, err error) {
	if m.err != nil {
		return 0, m.err
	}
	if offset >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n = copy(dst, m.data[offset:])
	if n < len(dst) {
		err = io.EOF
	}
	return n, err
}

func TestVectoredWriter_ReadFromAt(t *testing.T) {
	testErr := errors.New("reader at error")

	tests := []struct {
		name         string
		buffers      [][]byte
		reader       *mockReaderAt
		offset       int64
		maxSize      int64
		expectedN    int64
		expectedErr  error
		expectedBufs [][]byte
	}{
		{
			name: "ReadFromAt from zero offset fully",
			buffers: [][]byte{
				make([]byte, 3),
				make([]byte, 2),
			},
			reader:       &mockReaderAt{data: []byte("hello")},
			offset:       0,
			maxSize:      5,
			expectedN:    5,
			expectedErr:  nil,
			expectedBufs: [][]byte{[]byte("hel"), []byte("lo")},
		},
		{
			name: "ReadFromAt from non-zero offset across multiple buffers",
			buffers: [][]byte{
				make([]byte, 3),
				make([]byte, 3),
			},
			reader:       &mockReaderAt{data: []byte("0123456789")},
			offset:       3,
			maxSize:      6,
			expectedN:    6,
			expectedErr:  nil,
			expectedBufs: [][]byte{[]byte("345"), []byte("678")},
		},
		{
			name: "ReadFromAt with reader error",
			buffers: [][]byte{
				make([]byte, 5),
			},
			reader:       &mockReaderAt{err: testErr},
			offset:       0,
			maxSize:      5,
			expectedN:    0,
			expectedErr:  testErr,
			expectedBufs: [][]byte{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			pool := &TestBufferPool{Buffers: tc.buffers}
			w := NewVectoredWriter(pool, tc.maxSize)

			// Act
			n, err := w.ReadFromAt(tc.reader, tc.offset)

			// Assert
			assert.Equal(t, tc.expectedN, n)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedBufs, w.Buffers())
		})
	}
}

func TestVectoredWriter_Release(t *testing.T) {
	tests := []struct {
		name             string
		pool             *TestBufferPool
		maxSize          int64
		writeInput       []byte
		releaseCount     int
		expectedPutCount int
		expectedBufs     [][]byte
	}{
		{
			name: "release allocated buffers to pool and clear state",
			pool: &TestBufferPool{
				Buffers: [][]byte{
					make([]byte, 10),
					make([]byte, 10),
					make([]byte, 10),
				},
			},
			maxSize:          30,
			writeInput:       []byte("0123456789abcde"), // 15 bytes -> 2 buffers allocated
			releaseCount:     1,
			expectedPutCount: 2,
			expectedBufs:     [][]byte{},
		},
		{
			name:             "release with nil pool is safe",
			pool:             nil,
			maxSize:          100,
			writeInput:       nil,
			releaseCount:     1,
			expectedPutCount: 0,
			expectedBufs:     [][]byte{},
		},
		{
			name: "release with no allocated buffers is safe",
			pool: &TestBufferPool{
				Buffers: [][]byte{
					make([]byte, 10),
				},
			},
			maxSize:          10,
			writeInput:       nil, // 0 bytes -> 0 buffers allocated
			releaseCount:     1,
			expectedPutCount: 0,
			expectedBufs:     [][]byte{},
		},
		{
			name: "release multiple times is safe and idempotent",
			pool: &TestBufferPool{
				Buffers: [][]byte{
					make([]byte, 10),
				},
			},
			maxSize:          10,
			writeInput:       []byte("hello"), // 5 bytes -> 1 buffer allocated
			releaseCount:     2,
			expectedPutCount: 1,
			expectedBufs:     [][]byte{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			var pool BufferPool
			if tc.pool != nil {
				pool = tc.pool
			}
			w := NewVectoredWriter(pool, tc.maxSize)
			if len(tc.writeInput) > 0 {
				_, _ = w.Write(tc.writeInput)
			}

			// Act
			for i := 0; i < tc.releaseCount; i++ {
				w.Release()
			}

			// Assert
			if tc.pool != nil {
				assert.Equal(t, tc.expectedPutCount, len(tc.pool.PutBuffers))
			}
			assert.Equal(t, tc.expectedBufs, w.Buffers())
		})
	}
}

func TestVectoredWriter_OversizedBufferTruncation(t *testing.T) {
	// Arrange
	pool := &TestBufferPool{
		Buffers: [][]byte{
			make([]byte, 100),
		},
	}
	w := NewVectoredWriter(pool, 25)

	// Act
	n, err := w.Write([]byte("this is a test string of length greater than 25 characters"))

	// Assert
	assert.Equal(t, 25, n)
	assert.Equal(t, io.ErrShortWrite, err)
	assert.Equal(t, 1, len(w.Buffers()))
	assert.Equal(t, 25, len(w.Buffers()[0]))
}

func TestVectoredWriter_PoolExhaustion(t *testing.T) {
	tests := []struct {
		name        string
		isReadFrom  bool
		expectedN   int64
		expectedErr error
	}{
		{
			name:        "Write with pool exhaustion returns short write",
			isReadFrom:  false,
			expectedN:   0,
			expectedErr: io.ErrShortWrite,
		},
		{
			name:        "ReadFrom with pool exhaustion stops cleanly",
			isReadFrom:  true,
			expectedN:   0,
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			pool := &TestBufferPool{
				ReturnNilOnExhaustion: true,
			}
			w := NewVectoredWriter(pool, 100)

			// Act
			var n int64
			var err error
			if tc.isReadFrom {
				n, err = w.ReadFrom(bytes.NewReader([]byte("hello")))
			} else {
				var nw int
				nw, err = w.Write([]byte("hello"))
				n = int64(nw)
			}

			// Assert
			assert.Equal(t, tc.expectedN, n)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestVectoredWriter_ZeroOrNegativeMaxSize(t *testing.T) {
	tests := []struct {
		name    string
		maxSize int64
	}{
		{name: "zero maxSize", maxSize: 0},
		{name: "negative maxSize", maxSize: -10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			pool := &TestBufferPool{
				Buffers: [][]byte{make([]byte, 10)},
			}
			w := NewVectoredWriter(pool, tc.maxSize)

			// Act
			n, err := w.Write([]byte("hello"))

			// Assert
			assert.Equal(t, 0, n)
			assert.Equal(t, io.ErrShortWrite, err)
			assert.Empty(t, w.Buffers())
		})
	}
}

var globalWriter *VectoredWriter

func BenchmarkNewVectoredWriter(b *testing.B) {
	pool := &TestBufferPool{
		Buffers: [][]byte{
			make([]byte, 100),
			make([]byte, 100),
			make([]byte, 100),
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globalWriter = NewVectoredWriter(pool, 300)
	}
}
