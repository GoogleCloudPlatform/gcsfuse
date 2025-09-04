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
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/clock"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/fake"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	testutil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	var err error
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       100,
		Generation: 1234,
	}
	t.objectData = testutil.GenerateRandomBytes(int(t.object.Size))
	// Create the bucket.
	t.mockBucket = new(storage.TestifyMockBucket)
	t.mrdTimeout = time.Millisecond
	t.mrdWrapper, err = NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{WaitTime: t.mrdTimeout}, &cfg.Config{})
	assert.Nil(t.T(), err, "Error in creating MRDWrapper")
	t.mrdWrapper.Wrapped = fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond)
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
		{
			name:  "ReadEmpty",
			start: 10,
			end:   10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			buf := make([]byte, tc.end-tc.start)
			t.mrdWrapper.Wrapped = nil
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond))

			bytesRead, err := t.mrdWrapper.Read(context.Background(), buf, int64(tc.start), int64(tc.end), metrics.NewNoopMetrics())

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), tc.end-tc.start, bytesRead)
			assert.Equal(t.T(), t.objectData[tc.start:tc.end], buf[:bytesRead])
		})
	}
}

func (t *mrdWrapperTest) Test_Read_ErrorInCreatingMRD() {
	t.mrdWrapper.Wrapped = nil
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Error in creating MRD")).Once()

	bytesRead, err := t.mrdWrapper.Read(context.Background(), make([]byte, t.object.Size), 0, int64(t.object.Size), metrics.NewNoopMetrics())

	assert.ErrorContains(t.T(), err, "MultiRangeDownloaderWrapper::Read: Error in creating MultiRangeDownloader")
	assert.Equal(t.T(), 0, bytesRead)
}

func (t *mrdWrapperTest) TestReadContextCancelledWithInterruptsEnabled() {
	t.mrdWrapper.Wrapped = nil
	t.mrdWrapper.config = &cfg.Config{FileSystem: cfg.FileSystemConfig{IgnoreInterrupts: false}}
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond), nil).Once()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	bytesRead, err := t.mrdWrapper.Read(ctx, make([]byte, t.object.Size), 0, int64(t.object.Size), metrics.NewNoopMetrics())

	require.Error(t.T(), err)
	assert.ErrorContains(t.T(), err, "context canceled")
	assert.Equal(t.T(), 0, bytesRead)
}

func (t *mrdWrapperTest) TestReadContextCancelledWithInterruptsDisabled() {
	t.mrdWrapper.config = &cfg.Config{FileSystem: cfg.FileSystemConfig{IgnoreInterrupts: true}}
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond), nil).Once()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	bytesRead, err := t.mrdWrapper.Read(ctx, make([]byte, t.object.Size), 0, int64(t.object.Size), metrics.NewNoopMetrics())

	require.NoError(t.T(), err)
	assert.Equal(t.T(), 100, bytesRead)
}

func (t *mrdWrapperTest) Test_Read_EOF() {
	t.mrdWrapper.Wrapped = nil
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, t.objectData, time.Microsecond, io.EOF), nil).Once()

	_, err := t.mrdWrapper.Read(context.Background(), make([]byte, t.object.Size), 0, int64(t.object.Size), metrics.NewNoopMetrics())

	assert.ErrorIs(t.T(), err, io.EOF)
}

func (t *mrdWrapperTest) Test_Read_Error() {
	t.mrdWrapper.Wrapped = nil
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleepAndDefaultError(t.object, t.objectData, time.Microsecond, fmt.Errorf("Error")), nil).Once()

	bytesRead, err := t.mrdWrapper.Read(context.Background(), make([]byte, t.object.Size), 0, int64(t.object.Size), metrics.NewNoopMetrics())

	assert.ErrorContains(t.T(), err, "error in Add call")
	assert.Equal(t.T(), 0, bytesRead)
}

func (t *mrdWrapperTest) Test_NewMultiRangeDownloaderWrapper() {
	testCases := []struct {
		name   string
		bucket gcs.Bucket
		obj    *gcs.MinObject
		err    error
	}{
		{
			name:   "ValidParameters",
			bucket: t.mockBucket,
			obj:    t.object,
			err:    nil,
		},
		{
			name:   "NilMinObject",
			bucket: t.mockBucket,
			obj:    nil,
			err:    fmt.Errorf("NewMultiRangeDownloaderWrapperWithClock: Missing MinObject"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			_, err := NewMultiRangeDownloaderWrapper(tc.bucket, tc.obj, &cfg.Config{})
			if tc.err == nil {
				assert.NoError(t.T(), err)
			} else {
				assert.Error(t.T(), err)
				assert.EqualError(t.T(), err, tc.err.Error())
			}
		})
	}
}

func (t *mrdWrapperTest) Test_SetMinObject() {
	testCases := []struct {
		name string
		obj  *gcs.MinObject
		err  error
	}{
		{
			name: "ValidMinObject",
			obj:  t.object,
			err:  nil,
		},
		{
			name: "NilMinObject",
			obj:  nil,
			err:  fmt.Errorf("MultiRangeDownloaderWrapper::SetMinObject: Missing MinObject"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			err := t.mrdWrapper.SetMinObject(tc.obj)
			if tc.err == nil {
				assert.NoError(t.T(), err)
			} else {
				assert.Error(t.T(), err)
				assert.EqualError(t.T(), err, tc.err.Error())
			}
		})
	}
}

func (t *mrdWrapperTest) Test_EnsureMultiRangeDownloader() {
	testCases := []struct {
		name   string
		obj    *gcs.MinObject
		bucket gcs.Bucket
		err    error
	}{
		{
			name:   "ValidMinObject",
			obj:    t.object,
			bucket: t.mockBucket,
			err:    nil,
		},
		{
			name:   "NilMinObject",
			obj:    nil,
			bucket: t.mockBucket,
			err:    fmt.Errorf("ensureMultiRangeDownloader error: Missing minObject or bucket"),
		},
		{
			name:   "NilBucket",
			obj:    t.object,
			bucket: nil,
			err:    fmt.Errorf("ensureMultiRangeDownloader error: Missing minObject or bucket"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.mrdWrapper.bucket = tc.bucket
			t.mrdWrapper.object = tc.obj
			t.mrdWrapper.Wrapped = nil
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond))
			t.mrdWrapper.mu.RLock()
			defer t.mrdWrapper.mu.RUnlock()
			err := t.mrdWrapper.ensureMultiRangeDownloader(false)
			if tc.err == nil {
				assert.NoError(t.T(), err)
				assert.NotNil(t.T(), t.mrdWrapper.Wrapped)
			} else {
				assert.Error(t.T(), err)
				assert.EqualError(t.T(), err, tc.err.Error())
				assert.Nil(t.T(), t.mrdWrapper.Wrapped)
			}
		})
	}
}

func (t *mrdWrapperTest) Test_EnsureMultiRangeDownloader_UnusableExistingMRDTriggersRecreation() {
	t.mrdWrapper.bucket = t.mockBucket
	t.mrdWrapper.object = t.object
	t.mrdWrapper.Wrapped = fake.NewFakeMultiRangeDownloaderWithStatusError(t.object, t.objectData, fmt.Errorf("MRD is unusable..."))

	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, t.objectData, time.Microsecond))
	t.mrdWrapper.mu.RLock()
	defer t.mrdWrapper.mu.RUnlock()

	err := t.mrdWrapper.ensureMultiRangeDownloader(false)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), t.mrdWrapper.Wrapped)
	t.mockBucket.AssertExpectations(t.T())
}

func (t *mrdWrapperTest) Test_EnsureMultiRangeDownloader_UsableExistingMRDPreventsRecreation() {
	t.mrdWrapper.bucket = t.mockBucket
	t.mrdWrapper.object = t.object
	t.mrdWrapper.Wrapped = fake.NewFakeMultiRangeDownloaderWithStatusError(t.object, t.objectData, nil)
	t.mrdWrapper.mu.RLock()
	defer t.mrdWrapper.mu.RUnlock()

	err := t.mrdWrapper.ensureMultiRangeDownloader(false)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), t.mrdWrapper.Wrapped)
	t.mockBucket.AssertNotCalled(t.T(), "NewMultiRangeDownloader")
}
