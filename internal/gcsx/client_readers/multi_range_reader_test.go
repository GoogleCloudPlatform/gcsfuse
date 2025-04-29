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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/clock"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	testUtil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	TestTimeoutForMultiRangeRead = time.Second
)

type multiRangeReaderTest struct {
	suite.Suite
	ctx              context.Context
	object           *gcs.MinObject
	mockBucket       *storage.TestifyMockBucket
	multiRangeReader *MultiRangeReader
}

func TestMultiRangeReaderTestSuite(t *testing.T) {
	suite.Run(t, new(multiRangeReaderTest))
}

func (t *multiRangeReaderTest) SetupTest() {
	t.mockBucket = new(storage.TestifyMockBucket)
	t.ctx = context.Background()
	t.multiRangeReader = NewMultiRangeReader(common.NewNoopMetrics(), nil)
	t.object = &gcs.MinObject{
		Name:       testObject,
		Size:       17,
		Generation: 1234,
	}
}

func (t *multiRangeReaderTest) TearDown() {
	t.multiRangeReader.destroy()
}

func (t *multiRangeReaderTest) Test_ReadFromMultiRangeReader_ReadFull() {
	testCases := []struct {
		name      string
		dataSize  int
		extraSize int
	}{
		{
			name:      "ReadFull",
			dataSize:  100,
			extraSize: 0,
		},
		{
			name:      "ReadWithLargerBuffer",
			dataSize:  100,
			extraSize: 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.multiRangeReader.reader = nil
			t.multiRangeReader.isMRDInUse = false
			t.object.Size = uint64(tc.dataSize)
			testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
			fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{})
			require.NoError(t.T(), err, "Error in creating MRDWrapper")
			t.multiRangeReader.mrdWrapper = &fakeMRDWrapper
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond)).Times(1)
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Times(1)
			buf := make([]byte, tc.dataSize+tc.extraSize)

			bytesRead, err := t.multiRangeReader.readFromMultiRangeReader(t.ctx, buf, 0, int64(t.object.Size), TestTimeoutForMultiRangeRead)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), tc.dataSize, bytesRead)
			assert.Equal(t.T(), testContent[:tc.dataSize], buf[:bytesRead])
		})
	}
}

func (t *multiRangeReaderTest) Test_ReadFromMultiRangeReader_ValidateTimeout() {
	testCases := []struct {
		name               string
		dataSize           int
		timeout            time.Duration
		sleepTime          time.Duration
		expectedErrKeyword string
	}{
		{
			name:               "TimeoutPlusFiveMilliSecond",
			dataSize:           100,
			timeout:            5 * time.Millisecond,
			sleepTime:          10 * time.Millisecond,
			expectedErrKeyword: "Timeout",
		},
		{
			name:               "TimeoutValue",
			dataSize:           100,
			timeout:            5 * time.Millisecond,
			sleepTime:          5 * time.Millisecond,
			expectedErrKeyword: "Timeout",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.multiRangeReader.reader = nil
			t.multiRangeReader.isMRDInUse = false
			t.object.Size = uint64(tc.dataSize)
			testContent := testUtil.GenerateRandomBytes(int(t.object.Size))
			fakeMRDWrapper, err := gcsx.NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{})
			assert.Nil(t.T(), err, "Error in creating MRDWrapper")
			t.multiRangeReader.mrdWrapper = &fakeMRDWrapper
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, tc.sleepTime)).Once()
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Once()
			buf := make([]byte, tc.dataSize)

			bytesRead, err := t.multiRangeReader.readFromMultiRangeReader(t.ctx, buf, 0, int64(t.object.Size), tc.timeout)

			if tc.name == "TimeoutValue" && bytesRead != 0 {
				assert.NoError(t.T(), err)
				assert.Equal(t.T(), tc.dataSize, bytesRead)
				assert.Equal(t.T(), testContent[:tc.dataSize], buf[:bytesRead])
				return
			}
			assert.ErrorContains(t.T(), err, tc.expectedErrKeyword)
		})
	}
}
