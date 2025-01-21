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
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/clock"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type mrdWrapperTest struct {
	suite.Suite
	object     *gcs.MinObject
	objectData []byte
	mockBucket *storage.TestifyMockBucket
	mrdWrapper MultiRangeDownloaderWrapper
	mrdTimeout time.Duration
}

func TestMRDWrapperTestSuite(t *testing.T) {
	suite.Run(t, new(mrdWrapperTest))
}

func (t *mrdWrapperTest) SetupTest() {
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       100,
		Generation: 1234,
	}
	t.objectData = testutil.GenerateRandomBytes(int(t.object.Size))
	// Create the bucket.
	t.mockBucket = new(storage.TestifyMockBucket)
	t.mrdTimeout = time.Millisecond
	t.mrdWrapper = NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{WaitTime: t.mrdTimeout})
	t.mrdWrapper.Wrapped = fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond)
	t.mrdWrapper.refCount = 0
}

func (t *mrdWrapperTest) Test_IncrementRefCount_ParallelUpdates() {
	const finalRefCount int = 1
	wg := sync.WaitGroup{}
	for i := 0; i < finalRefCount; i++ {
		wg.Add(1)
		go func() {
			t.mrdWrapper.IncrementRefCount()
			wg.Done()
		}()
	}
	wg.Wait()

	assert.Equal(t.T(), finalRefCount, t.mrdWrapper.refCount)
}

// func (t *mrdWrapperTest) Test_IncrementRefCount_CancelCleanup() {
// 	const finalRefCount int = 1
// 	t.mrdWrapper.IncrementRefCount()
// 	err := t.mrdWrapper.DecrementRefCount()

// 	assert.Nil(t.T(), err)
// 	assert.NotNil(t.T(), t.mrdWrapper.cancelCleanup)
// 	assert.NotNil(t.T(), t.mrdWrapper.Wrapped)

// 	t.mrdWrapper.IncrementRefCount()

// 	assert.Equal(t.T(), finalRefCount, t.mrdWrapper.refCount)
// 	assert.Nil(t.T(), t.mrdWrapper.cancelCleanup)
// 	assert.NotNil(t.T(), t.mrdWrapper.Wrapped)
// }

func (t *mrdWrapperTest) Test_DecrementRefCount_ParallelUpdates() {
	const finalRefCount int = 0
	maxRefCount := 10
	wg := sync.WaitGroup{}
	// Incrementing refcount in parallel.
	for i := 0; i < maxRefCount; i++ {
		wg.Add(1)
		go func() {
			t.mrdWrapper.IncrementRefCount()
			wg.Done()
		}()
	}
	wg.Wait()
	// Decrementing refcount in parallel.
	for i := 0; i < maxRefCount; i++ {
		wg.Add(1)
		go func() {
			err := t.mrdWrapper.DecrementRefCount()
			assert.Nil(t.T(), err)
			wg.Done()
		}()
	}
	wg.Wait()

	assert.Equal(t.T(), finalRefCount, t.mrdWrapper.GetRefCount())
	// assert.NotNil(t.T(), t.mrdWrapper.Wrapped)
	// assert.NotNil(t.T(), t.mrdWrapper.cancelCleanup)
	// Waiting for the cleanup to be done.
	// time.Sleep(t.mrdTimeout + time.Millisecond)
	assert.Nil(t.T(), t.mrdWrapper.Wrapped)
}

func (t *mrdWrapperTest) Test_DecrementRefCount_InvalidUse() {
	errMsg := "MultiRangeDownloaderWrapper DecrementRefCount: Refcount cannot be negative"
	assert.ErrorContains(t.T(), t.mrdWrapper.DecrementRefCount(), errMsg)
}

func (t *mrdWrapperTest) Test_Read() {
	testCases := []struct {
		name  string
		start int
		end   int
	}{
		{
			name:  "ReadFull",
			start: 0,
			end:   int(t.object.Size),
		},
		{
			name:  "ReadChunk",
			start: 10,
			end:   10 + int(t.object.Size)/2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			buf := make([]byte, tc.end-tc.start)
			t.mrdWrapper.Wrapped = nil
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond))

			bytesRead, err := t.mrdWrapper.Read(context.Background(), buf, int64(tc.start), int64(tc.end), time.Millisecond, common.NewNoopMetrics())

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), tc.end-tc.start, bytesRead)
			assert.Equal(t.T(), t.objectData[tc.start:tc.end], buf[:bytesRead])
		})
	}
}
