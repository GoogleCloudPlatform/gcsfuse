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
	"bytes"
	"io"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const fakeHandleData = "fake-handle"

type rangeReaderTest struct {
	suite.Suite
	rangeReader *RangeReader
}

func TestRangeReaderTestSuite(t *testing.T) {
	suite.Run(t, new(rangeReaderTest))
}

func (t *rangeReaderTest) SetupTest() {
	t.rangeReader = NewRangeReader()
}

func (t *rangeReaderTest) TearDown() {
	t.rangeReader.Destroy()
}

func getReadCloser(content []byte) io.ReadCloser {
	r := bytes.NewReader(content)
	rc := io.NopCloser(r)
	return rc
}

func getReader() *fake.FakeReader {
	testContent := testutil.GenerateRandomBytes(2)
	return &fake.FakeReader{
		ReadCloser: getReadCloser(testContent),
		Handle:     []byte(fakeHandleData),
	}
}

func (t *rangeReaderTest) Test_NewRangeReader() {
	// rangeReader is instantiated in setup.
	assert.Equal(t.T(), int64(-1), t.rangeReader.start)
	assert.Equal(t.T(), int64(-1), t.rangeReader.limit)
}

func (t *rangeReaderTest) Test_CheckInvariants() {
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
				t.rangeReader.reader = getReader()
				return &RangeReader{
					start:  0,
					limit:  10,
					reader: t.rangeReader.reader,
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
			name: "negative limit with valid reader",
			setup: func() *RangeReader {
				t.rangeReader.reader = getReader()
				return &RangeReader{
					start:  -10,
					limit:  -5,
					reader: t.rangeReader.reader,
					cancel: func() {},
				}
			},
			shouldPanic: true,
		},
		{
			name: "negative limit with nil reader",
			setup: func() *RangeReader {
				return &RangeReader{
					start:  -10,
					limit:  -5,
					reader: nil,
					cancel: nil,
				}
			},
			shouldPanic: false,
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

func (t *rangeReaderTest) Test_Destroy_NonNilReader() {
	t.rangeReader.reader = getReader()

	t.rangeReader.Destroy()

	assert.Nil(t.T(), t.rangeReader.Reader)
	assert.Nil(t.T(), t.rangeReader.cancel)
	assert.Equal(t.T(), []byte(fakeHandleData), t.rangeReader.readHandle)
}
