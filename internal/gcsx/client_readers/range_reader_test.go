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

package gcsx

import (
	"context"
	"errors"
	"testing"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type rangeReaderTest struct {
	suite.Suite
	ctx    context.Context
	reader gcs.StorageReader
}

func TestRangeReaderReaderTestSuite(t *testing.T) {
	suite.Run(t, new(rangeReaderTest))
}

type MockStorageReader struct {
	gcs.StorageReader
	mock.Mock
	readHandle []byte
}

func (m *MockStorageReader) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (r *MockStorageReader) ReadHandle() storagev2.ReadHandle {
	return r.readHandle
}

func (t *rangeReaderTest) SetupTest() {
	t.reader = new(MockStorageReader)
}

func (t *rangeReaderTest) TestRangeReader_CheckInvariants() {
	tests := []struct {
		name        string
		setup       func() *RangeReader
		shouldPanic bool
	}{
		{
			name: "valid no reader",
			setup: func() *RangeReader {
				return &RangeReader{
					start:  0,
					limit:  10,
					reader: nil,
					cancel: nil,
				}
			},
			shouldPanic: false,
		},
		{
			name: "reader without cancel",
			setup: func() *RangeReader {
				return &RangeReader{
					start:  0,
					limit:  10,
					reader: t.reader,
					cancel: nil,
				}
			},
			shouldPanic: true,
		},
		{
			name: "cancel without reader",
			setup: func() *RangeReader {
				return &RangeReader{
					start:  0,
					limit:  10,
					reader: nil,
					cancel: func() {},
				}
			},
			shouldPanic: true,
		},
		{
			name: "invalid range",
			setup: func() *RangeReader {
				return &RangeReader{
					start:  20,
					limit:  10,
					reader: nil,
					cancel: nil,
				}
			},
			shouldPanic: true,
		},
		{
			name: "negative limit with nil reader",
			setup: func() *RangeReader {
				return &RangeReader{
					start:  0,
					limit:  -1,
					reader: nil,
					cancel: nil,
				}
			},
			shouldPanic: true,
		},
		{
			name: "negative limit with valid reader",
			setup: func() *RangeReader {
				return &RangeReader{
					start:  0,
					limit:  -5,
					reader: t.reader,
					cancel: func() {},
				}
			},
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func() {
			rr := tt.setup()
			if tt.shouldPanic {
				assert.Panics(t.T(), func() { rr.CheckInvariants() }, "Expected panic")
			} else {
				assert.NotPanics(t.T(), func() { rr.CheckInvariants() }, "Expected no panic")
			}
		})
	}
}

func (t *rangeReaderTest) TestRangeReader_Destroy() {
	mockReader := &MockStorageReader{}
	mockReader.readHandle = []byte("test-handle")
	mockReader.On("Close").Return(nil)
	cancel := func() {}
	rr := &RangeReader{
		reader: mockReader,
		cancel: cancel,
	}

	rr.Destroy()

	assert.Nil(t.T(), rr.Reader)
	assert.Nil(t.T(), rr.cancel)
	assert.Equal(t.T(), []byte("test-handle"), rr.readHandle)
	mockReader.AssertCalled(t.T(), "Close")
}

func TestRangeReader_closeReader(t *testing.T) {
	tests := []struct {
		name           string
		closeError     error
		expectedHandle []byte
	}{
		{
			name:           "successful close",
			closeError:     nil,
			expectedHandle: []byte("abc123"),
		},
		{
			name:           "close with error",
			closeError:     errors.New("something went wrong"),
			expectedHandle: []byte("xyz456"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReader := &MockStorageReader{}
			mockReader.readHandle = tt.expectedHandle
			mockReader.On("Close").Return(tt.closeError)

			rr := &RangeReader{
				reader: mockReader,
			}

			rr.closeReader()

			assert.Equal(t, tt.expectedHandle, rr.readHandle)
			mockReader.AssertCalled(t, "Close")
		})
	}
}
