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
	"context"
	"io"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type rangeReaderTest struct {
	suite.Suite
	ctx         context.Context
	rangeReader RangeReader
}

func TestRangeReaderReaderTestSuite(t *testing.T) {
	suite.Run(t, new(rangeReaderTest))
}

func (t *rangeReaderTest) SetupTest() {
}

func getReadCloser(content []byte) io.ReadCloser {
	r := bytes.NewReader(content)
	rc := io.NopCloser(r)
	return rc
}

func (t *rangeReaderTest) TestRangeReader_Destroy_NonNilReader() {
	testContent := testutil.GenerateRandomBytes(2)
	rc := &fake.FakeReader{
		ReadCloser: getReadCloser(testContent),
		Handle:     []byte("fake-handle"),
	}
	t.rangeReader.reader = rc

	t.rangeReader.Destroy()

	assert.Nil(t.T(), t.rangeReader.Reader)
	assert.Nil(t.T(), t.rangeReader.cancel)
	assert.Equal(t.T(), []byte("fake-handle"), t.rangeReader.readHandle)
}
