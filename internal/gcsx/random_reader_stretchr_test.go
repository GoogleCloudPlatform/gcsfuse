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
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
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

const TestTimeoutForMultiRangeRead = time.Second

type RandomReaderStretchrTest struct {
	suite.Suite
	object       *gcs.MinObject
	mockBucket   *storage.TestifyMockBucket
	rr           checkingRandomReader
	cacheDir     string
	jobManager   *downloader.JobManager
	cacheHandler *file.CacheHandler
}

func TestRandomReaderStretchrTestSuite(t *testing.T) {
	suite.Run(t, new(RandomReaderStretchrTest))
}

func (t *RandomReaderStretchrTest) SetupTest() {
	t.rr.ctx = context.Background()

	// Manufacture an object record.
	t.object = &gcs.MinObject{
		Name:       "foo",
		Size:       17,
		Generation: 1234,
	}

	// Create the bucket.
	t.mockBucket = new(storage.TestifyMockBucket)

	t.cacheDir = path.Join(os.Getenv("HOME"), "cache/dir")
	lruCache := lru.NewCache(cacheMaxSize)
	fileCacheConfig := &cfg.FileCacheConfig{
		EnableCrc: false,
	}
	t.jobManager = downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, t.cacheDir, sequentialReadSizeInMb, fileCacheConfig, nil)
	t.cacheHandler = file.NewCacheHandler(lruCache, t.jobManager, t.cacheDir, util.DefaultFilePerm, util.DefaultDirPerm, "", "", false)

	// Set up the reader.
	rr := NewRandomReader(t.object, t.mockBucket, sequentialReadSizeInMb, nil, false, metrics.NewNoopMetrics(), nil, nil)
	t.rr.wrapped = rr.(*randomReader)
}

func (t *RandomReaderStretchrTest) TearDownTest() {
	t.rr.Destroy()
}

func (t *RandomReaderStretchrTest) Test_GetReadInfo() {
	testCases := []struct {
		name                  string
		offset                int64
		seekRecorded          bool
		initialReadType       int64
		initialExpOffset      int64
		initialNumSeeks       uint64
		initialTotalReadBytes uint64
		expectedReadType      int64
		expectedNumSeeks      uint64
	}{
		{
			name:                  "First Read",
			offset:                0,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      0,
			initialNumSeeks:       0,
			initialTotalReadBytes: 0,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      0,
		},
		{
			name:                  "Sequential Read",
			offset:                10,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      10,
			initialNumSeeks:       0,
			initialTotalReadBytes: 100,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      0,
		},
		{
			name:                  "Sequential read with small forward jump and high average read bytes is still sequential",
			offset:                100,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      10,
			initialNumSeeks:       0,
			initialTotalReadBytes: 10000000,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      0,
		},
		{
			name:                  "Sequential read with large forward jump is a seek",
			offset:                50 + maxReadSize + 1,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       0,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      1,
		},
		{
			name:                  "Sequential read with backward jump is a seek",
			offset:                49,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       0,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      1,
		},
		{
			name:                  "Contiguous random read is not a seek",
			offset:                50,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeRandom,
			initialExpOffset:      50,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      minSeeksForRandom,
		},
		{
			name:                  "Non-contiguous random read is a seek",
			offset:                100,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeRandom,
			initialExpOffset:      50,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      minSeeksForRandom + 1,
		},
		{
			name:                  "Switches to random read after enough seeks",
			offset:                50 + maxReadSize + 1,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       minSeeksForRandom - 1,
			initialTotalReadBytes: 1000,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      minSeeksForRandom,
		},
		{
			name:                  "Switches back to sequential with high average read bytes",
			offset:                100,
			seekRecorded:          false,
			initialReadType:       metrics.ReadTypeRandom,
			initialExpOffset:      50,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: maxReadSize * (minSeeksForRandom + 1),
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      minSeeksForRandom + 1,
		},
		{
			name:                  "Seek recorded: sequential large forward jump",
			offset:                50 + maxReadSize + 1,
			seekRecorded:          true,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       0,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      0, // Not incremented
		},
		{
			name:                  "Seek recorded: sequential backward jump",
			offset:                49,
			seekRecorded:          true,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       1,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeSequential,
			expectedNumSeeks:      1, // Not incremented
		},
		{
			name:                  "Seek recorded: non-contiguous random read",
			offset:                100,
			seekRecorded:          true,
			initialReadType:       metrics.ReadTypeRandom,
			initialExpOffset:      50,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: 50 * 1024,
			expectedReadType:      metrics.ReadTypeRandom,
			expectedNumSeeks:      minSeeksForRandom, // Not incremented
		},
		{
			name:                  "Seek recorded: does not switch to random",
			offset:                50 + maxReadSize + 1,
			seekRecorded:          true,
			initialReadType:       metrics.ReadTypeSequential,
			initialExpOffset:      50,
			initialNumSeeks:       minSeeksForRandom - 1,
			initialTotalReadBytes: 1000,
			expectedReadType:      metrics.ReadTypeSequential, // Does not switch
			expectedNumSeeks:      minSeeksForRandom - 1,      // Not incremented
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			rr := &randomReader{}
			rr.readType.Store(tc.initialReadType)
			rr.expectedOffset.Store(tc.initialExpOffset)
			rr.seeks.Store(tc.initialNumSeeks)
			rr.totalReadBytes.Store(tc.initialTotalReadBytes)

			readInfo := rr.getReadInfo(tc.offset, tc.seekRecorded)
			assert.Equal(t.T(), tc.expectedReadType, readInfo.readType, "Read type mismatch")
			assert.Equal(t.T(), tc.expectedNumSeeks, rr.seeks.Load(), "Number of seeks mismatch")
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadAt_ParallelMRDReads() {
	// Setup
	t.rr.wrapped.reader = nil
	t.rr.wrapped.seeks.Store(minSeeksForRandom)
	t.rr.wrapped.readType.Store(metrics.ReadTypeRandom)
	t.object.Size = 20 * MiB
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))

	// Mock bucket and MRD
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true})
	fakeMRDWrapper, err := NewMultiRangeDownloaderWrapper(t.mockBucket, t.object, &cfg.Config{})
	require.NoError(t.T(), err)
	t.rr.wrapped.mrdWrapper = &fakeMRDWrapper
	t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloader(t.object, testContent), nil)

	// Parallel reads
	tasks := []struct {
		offset int64
		size   int
	}{
		{0, 1 * MiB},
		{2 * MiB, 2 * MiB},
		{5 * MiB, 1 * MiB},
		{10 * MiB, 5 * MiB},
	}

	var wg sync.WaitGroup
	var totalBytesReadFromTasks uint64

	for _, task := range tasks {
		wg.Add(1)
		totalBytesReadFromTasks += uint64(task.size)
		go func(offset int64, size int) {
			defer wg.Done()
			buf := make([]byte, size)
			// Each goroutine gets its own context.
			ctx := context.Background()
			objData, err := t.rr.wrapped.ReadAt(ctx, buf, offset)

			require.NoError(t.T(), err)
			require.Equal(t.T(), size, objData.Size)
			require.Equal(t.T(), testContent[offset:offset+int64(size)], buf)
		}(task.offset, task.size)
	}

	wg.Wait()

	// Validation
	assert.Equal(t.T(), totalBytesReadFromTasks, t.rr.wrapped.totalReadBytes.Load())
	assert.Equal(t.T(), 1, t.rr.wrapped.mrdWrapper.GetRefCount())
	assert.True(t.T(), t.rr.wrapped.isMRDInUse.Load())
}

func (t *RandomReaderStretchrTest) Test_ReaderType() {
	testCases := []struct {
		name       string
		readType   int64
		start      int64
		end        int64
		bucketType gcs.BucketType
		readerType ReaderType
	}{
		{
			name:       "ZonalBucketRandomRead",
			readType:   metrics.ReadTypeRandom,
			start:      50,
			end:        68,
			bucketType: gcs.BucketType{Zonal: true},
			readerType: MultiRangeReader,
		},
		{
			name:       "ZonalBucketSequentialRead",
			readType:   metrics.ReadTypeSequential,
			start:      50,
			end:        68,
			bucketType: gcs.BucketType{Zonal: true},
			readerType: RangeReader,
		},
		{
			name:       "RegularBucketRandomRead",
			readType:   metrics.ReadTypeRandom,
			start:      50,
			end:        68,
			bucketType: gcs.BucketType{Zonal: false},
			readerType: RangeReader,
		},
		{
			name:       "RegularBucketSequentialRead",
			readType:   metrics.ReadTypeSequential,
			start:      50,
			end:        68,
			bucketType: gcs.BucketType{Zonal: false},
			readerType: RangeReader,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			readerType := readerType(tc.readType, tc.bucketType)
			assert.Equal(t.T(), readerType, tc.readerType)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_GetEndOffset() {
	testCases := []struct {
		name                  string
		start                 int64
		objectSize            int64
		initialReadType       int64
		initialNumSeeks       uint64
		initialTotalReadBytes uint64
		sequentialReadSizeMb  int32
		expectedEnd           int64
	}{
		{
			name:                  "Sequential Read, Fits in sequentialReadSizeMb",
			start:                 0,
			objectSize:            10 * MiB,
			initialReadType:       metrics.ReadTypeSequential,
			initialNumSeeks:       0,
			initialTotalReadBytes: 0,
			sequentialReadSizeMb:  22,
			expectedEnd:           10 * MiB,
		},
		{
			name:                  "Sequential Read, Object Larger than sequentialReadSizeMb",
			start:                 0,
			objectSize:            50 * MiB,
			initialReadType:       metrics.ReadTypeSequential,
			initialNumSeeks:       0,
			initialTotalReadBytes: 0,
			sequentialReadSizeMb:  22,
			expectedEnd:           22 * MiB,
		},
		{
			name:                  "Sequential Read, Respects object size",
			start:                 5 * MiB,
			objectSize:            7 * MiB,
			initialReadType:       metrics.ReadTypeSequential,
			initialNumSeeks:       0,
			initialTotalReadBytes: 0,
			sequentialReadSizeMb:  22,
			expectedEnd:           7 * MiB,
		},
		{
			name:                  "Random Read, Min read size",
			start:                 0,
			objectSize:            5 * MiB,
			initialReadType:       metrics.ReadTypeRandom,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: 1000,
			sequentialReadSizeMb:  22,
			expectedEnd:           minReadSize,
		},
		{
			name:                  "Random Read, Averages less than minReadSize",
			start:                 0,
			objectSize:            50 * MiB,
			initialReadType:       metrics.ReadTypeRandom,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: 100 * 1024, // 100KiB
			sequentialReadSizeMb:  22,
			expectedEnd:           minReadSize, // Should be atleast minReadSize
		},
		{
			name:                  "Random Read, Start Offset Non-Zero",
			start:                 5 * MiB,
			objectSize:            50 * MiB,
			initialReadType:       metrics.ReadTypeRandom,
			initialNumSeeks:       minSeeksForRandom,
			initialTotalReadBytes: 2 * MiB, // avg read bytes = 1MiB
			sequentialReadSizeMb:  22,
			expectedEnd:           5*MiB + 2*MiB, // avg read bytes + 1MiB
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			rr := &randomReader{
				object:               &gcs.MinObject{Size: uint64(tc.objectSize)},
				sequentialReadSizeMb: tc.sequentialReadSizeMb,
			}
			rr.readType.Store(tc.initialReadType)
			rr.seeks.Store(tc.initialNumSeeks)
			rr.totalReadBytes.Store(tc.initialTotalReadBytes)

			end := rr.getEndOffset(tc.start)

			assert.Equal(t.T(), tc.expectedEnd, end, "End offset mismatch")
		})
	}
}

func (t *RandomReaderStretchrTest) Test_IsSeekNeeded() {
	testCases := []struct {
		name           string
		readType       int64
		offset         int64
		expectedOffset int64
		want           bool
	}{
		{
			name:           "First read, expectedOffset is 0",
			readType:       metrics.ReadTypeSequential,
			offset:         100,
			expectedOffset: 0,
			want:           false,
		},
		{
			name:           "Random read, same offset",
			readType:       metrics.ReadTypeRandom,
			offset:         100,
			expectedOffset: 100,
			want:           false,
		},
		{
			name:           "Random read, different offset",
			readType:       metrics.ReadTypeRandom,
			offset:         200,
			expectedOffset: 100,
			want:           true,
		},
		{
			name:           "Sequential read, same offset",
			readType:       metrics.ReadTypeSequential,
			offset:         100,
			expectedOffset: 100,
			want:           false,
		},
		{
			name:           "Sequential read, small forward jump within maxReadSize",
			readType:       metrics.ReadTypeSequential,
			offset:         100 + maxReadSize/2,
			expectedOffset: 100,
			want:           false,
		},
		{
			name:           "Sequential read, forward jump to boundary of maxReadSize",
			readType:       metrics.ReadTypeSequential,
			offset:         100 + maxReadSize,
			expectedOffset: 100,
			want:           false,
		},
		{
			name:           "Sequential read, large forward jump beyond maxReadSize",
			readType:       metrics.ReadTypeSequential,
			offset:         100 + maxReadSize + 1,
			expectedOffset: 100,
			want:           true,
		},
		{
			name:           "Sequential read, backward jump",
			readType:       metrics.ReadTypeSequential,
			offset:         99,
			expectedOffset: 100,
			want:           true,
		},
		{
			name:           "Unknown read type",
			readType:       -1, // An invalid read type
			offset:         200,
			expectedOffset: 100,
			want:           false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			got := isSeekNeeded(tc.readType, tc.offset, tc.expectedOffset)
			assert.Equal(t.T(), tc.want, got)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromRangeReader_WhenExistingReaderIsNil() {
	testCases := []struct {
		name             string
		inputReadHandle  []byte
		outputReadHandle []byte
	}{
		{
			name:            "ReadHandlePresent",
			inputReadHandle: []byte("fake-handle"),
		},
		{
			name:            "ReadHandleAbsent",
			inputReadHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rr.wrapped.readHandle = tc.inputReadHandle
			t.rr.wrapped.reader = nil
			t.rr.wrapped.start = 0
			t.object.Size = 5
			dataSize := 5
			testContent := testutil.GenerateRandomBytes(int(t.object.Size))
			rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
			readObjectRequest := &gcs.ReadObjectRequest{
				Name:       t.rr.wrapped.object.Name,
				Generation: t.rr.wrapped.object.Generation,
				Range: &gcs.ByteRange{
					Start: uint64(0),
					Limit: t.object.Size,
				},
				ReadCompressed: t.rr.wrapped.object.HasContentEncodingGzip(),
				ReadHandle:     t.rr.wrapped.readHandle,
			}
			t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, nil).Times(1)
			buf := make([]byte, dataSize)

			n, err := t.rr.wrapped.readFromRangeReader(t.rr.ctx, buf, 0, int64(t.object.Size), metrics.ReadTypeUnknown)

			t.mockBucket.AssertExpectations(t.T())
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), dataSize, n)
			assert.Equal(t.T(), testContent[:dataSize], buf)
			// Verify the reader state.
			assert.Nil(t.T(), t.rr.wrapped.reader)
			assert.Nil(t.T(), t.rr.wrapped.cancel)
			assert.Equal(t.T(), int64(5), t.rr.wrapped.start)
			assert.Equal(t.T(), int64(5), t.rr.wrapped.limit)
			assert.Equal(t.T(), int64(t.object.Size), t.rr.wrapped.expectedOffset.Load())
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromRangeReader_WhenExistingReaderIsNotNil() {
	t.rr.wrapped.start = 4
	t.rr.wrapped.limit = 10
	t.rr.wrapped.totalReadBytes.Store(4)
	t.object.Size = 10
	dataSize := 4
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.rr.wrapped.reader = rc
	t.rr.wrapped.cancel = func() {}
	buf := make([]byte, dataSize)

	n, err := t.rr.wrapped.readFromRangeReader(t.rr.ctx, buf, 4, 8, metrics.ReadTypeUnknown)

	assert.NoError(t.T(), err)
	assert.Equal(t.T(), dataSize, n)
	// Verify the reader state.
	assert.Equal(t.T(), rc, t.rr.wrapped.reader)
	assert.NotNil(t.T(), t.rr.wrapped.cancel)
	assert.Equal(t.T(), int64(8), t.rr.wrapped.start)
	assert.Equal(t.T(), int64(10), t.rr.wrapped.limit)
	assert.Equal(t.T(), uint64(8), t.rr.wrapped.totalReadBytes.Load())
	assert.Equal(t.T(), int64(8), t.rr.wrapped.expectedOffset.Load())
}

func (t *RandomReaderStretchrTest) Test_ReadFromRangeReader_WhenAllDataFromReaderIsRead() {
	testCases := []struct {
		name       string
		readHandle []byte
	}{
		{
			name:       "GCSReturnedReadHandle",
			readHandle: []byte("fake-handle"),
		},
		{
			name:       "GCSReturnedNoReadHandle",
			readHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rr.wrapped.start = 4
			t.rr.wrapped.limit = 10
			t.rr.wrapped.totalReadBytes.Store(4)
			t.object.Size = 10
			dataSize := 6
			testContent := testutil.GenerateRandomBytes(int(t.object.Size))
			rc := &fake.FakeReader{
				ReadCloser: getReadCloser(testContent),
				Handle:     tc.readHandle,
			}
			t.rr.wrapped.reader = rc
			t.rr.wrapped.cancel = func() {}
			buf := make([]byte, dataSize)

			n, err := t.rr.wrapped.readFromRangeReader(t.rr.ctx, buf, 4, 10, metrics.ReadTypeUnknown)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), dataSize, n)
			// Verify the reader state.
			assert.Nil(t.T(), t.rr.wrapped.reader)
			assert.Nil(t.T(), t.rr.wrapped.cancel)
			assert.Equal(t.T(), int64(10), t.rr.wrapped.start)
			assert.Equal(t.T(), int64(10), t.rr.wrapped.limit)
			assert.Equal(t.T(), uint64(10), t.rr.wrapped.totalReadBytes.Load())
			assert.Equal(t.T(), int64(10), t.rr.wrapped.expectedOffset.Load())
			expectedReadHandle := tc.readHandle
			assert.Equal(t.T(), expectedReadHandle, t.rr.wrapped.readHandle)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromRangeReader_WhenReaderHasLessDataThanRequested() {
	testCases := []struct {
		name       string
		readHandle []byte
	}{
		{
			name:       "GCSReturnedReadHandle",
			readHandle: []byte("fake-handle"),
		},
		{
			name:       "GCSReturnedNoReadHandle",
			readHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rr.wrapped.start = 0
			t.rr.wrapped.limit = 6
			t.rr.wrapped.totalReadBytes.Store(0)
			dataSize := 6
			testContent := testutil.GenerateRandomBytes(dataSize)
			rc := &fake.FakeReader{
				ReadCloser: getReadCloser(testContent),
				Handle:     tc.readHandle,
			}
			t.rr.wrapped.reader = rc
			t.rr.wrapped.cancel = func() {}
			buf := make([]byte, 10)

			n, err := t.rr.wrapped.readFromRangeReader(t.rr.ctx, buf, 0, 10, metrics.ReadTypeUnknown)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), dataSize, n)
			// Verify the reader state.
			assert.Nil(t.T(), t.rr.wrapped.reader)
			assert.Nil(t.T(), t.rr.wrapped.cancel)
			assert.Equal(t.T(), int64(dataSize), t.rr.wrapped.start)
			assert.Equal(t.T(), int64(dataSize), t.rr.wrapped.limit)
			assert.Equal(t.T(), uint64(dataSize), t.rr.wrapped.totalReadBytes.Load())
			assert.Equal(t.T(), int64(dataSize), t.rr.wrapped.expectedOffset.Load())
			expectedReadHandle := tc.readHandle
			assert.Equal(t.T(), expectedReadHandle, t.rr.wrapped.readHandle)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromRangeReader_WhenReaderReturnedMoreData() {
	testCases := []struct {
		name       string
		readHandle []byte
	}{
		{
			name:       "GCSReturnedReadHandle",
			readHandle: []byte("fake-handle"),
		},
		{
			name:       "GCSReturnedNoReadHandle",
			readHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rr.wrapped.start = 0
			t.rr.wrapped.limit = 6
			t.rr.wrapped.totalReadBytes.Store(0)
			dataSize := 8
			testContent := testutil.GenerateRandomBytes(dataSize)
			rc := &fake.FakeReader{
				ReadCloser: getReadCloser(testContent),
				Handle:     tc.readHandle,
			}
			t.rr.wrapped.reader = rc
			t.rr.wrapped.cancel = func() {}
			buf := make([]byte, 10)

			_, err := t.rr.wrapped.readFromRangeReader(t.rr.ctx, buf, 0, 10, metrics.ReadTypeUnknown)

			assert.True(t.T(), strings.Contains(err.Error(), "extra bytes: 2"))
			assert.Nil(t.T(), t.rr.wrapped.reader)
			assert.Nil(t.T(), t.rr.wrapped.cancel)
			assert.Equal(t.T(), int64(-1), t.rr.wrapped.start)
			assert.Equal(t.T(), int64(-1), t.rr.wrapped.limit)
			assert.Equal(t.T(), uint64(8), t.rr.wrapped.totalReadBytes.Load())
			assert.Equal(t.T(), int64(0), t.rr.wrapped.expectedOffset.Load())
			expectedReadHandle := tc.readHandle
			assert.Equal(t.T(), expectedReadHandle, t.rr.wrapped.readHandle)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromRangeReader_WhenReaderReturnedEOF() {
	t.rr.wrapped.start = 0
	t.rr.wrapped.limit = 10
	dataSize := 6
	testContent := testutil.GenerateRandomBytes(dataSize)
	rc := &fake.FakeReader{ReadCloser: getReadCloser(testContent)}
	t.rr.wrapped.reader = rc
	t.rr.wrapped.cancel = func() {}
	buf := make([]byte, 10)

	_, err := t.rr.wrapped.readFromRangeReader(t.rr.ctx, buf, 0, 10, metrics.ReadTypeUnknown)

	assert.True(t.T(), strings.Contains(err.Error(), "skipping 4 bytes"))
	assert.Equal(t.T(), int64(0), t.rr.wrapped.expectedOffset.Load())
}

func (t *RandomReaderStretchrTest) Test_ExistingReader_WrongOffset() {
	testCases := []struct {
		name       string
		readHandle []byte
	}{
		{
			name:       "ReaderHasReadHandle",
			readHandle: []byte("fake-handle"),
		},
		{
			name:       "ReaderHasNoReadHandle",
			readHandle: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			// Simulate an existing reader.
			t.rr.wrapped.readHandle = tc.readHandle
			t.rr.wrapped.reader = &fake.FakeReader{
				ReadCloser: io.NopCloser(strings.NewReader("xxx")),
				Handle:     tc.readHandle,
			}
			t.rr.wrapped.cancel = func() {}
			t.rr.wrapped.start = 2
			t.rr.wrapped.limit = 5
			readObjectRequest := &gcs.ReadObjectRequest{
				Name:       t.rr.wrapped.object.Name,
				Generation: t.rr.wrapped.object.Generation,
				Range: &gcs.ByteRange{
					Start: uint64(0),
					Limit: t.object.Size,
				},
				ReadCompressed: t.rr.wrapped.object.HasContentEncodingGzip(),
				ReadHandle:     t.rr.wrapped.readHandle,
			}
			t.mockBucket.
				On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).
				Return(nil, errors.New(string(tc.readHandle))).
				Times(1)
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(2)

			buf := make([]byte, 1)

			_, err := t.rr.ReadAt(buf, 0)

			t.mockBucket.AssertExpectations(t.T())
			assert.NotNil(t.T(), err)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadAt_ExistingReaderLimitIsLessThanRequestedDataSize() {
	t.object.Size = 10
	// Simulate an existing reader.
	t.rr.wrapped.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte("xxx")), Handle: []byte("fake")}
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 2
	t.rr.wrapped.limit = 5
	rc := &fake.FakeReader{ReadCloser: getReadCloser([]byte("abcdefgh"))}
	expectedHandleInRequest := []byte(t.rr.wrapped.reader.ReadHandle())
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       t.rr.wrapped.object.Name,
		Generation: t.rr.wrapped.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(2),
			Limit: t.object.Size,
		},
		ReadCompressed: t.rr.wrapped.object.HasContentEncodingGzip(),
		ReadHandle:     expectedHandleInRequest,
	}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, nil)
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(2)
	requestSize := 6
	buf := make([]byte, requestSize)

	data, err := t.rr.ReadAt(buf, 2)

	require.Nil(t.T(), err)
	require.Equal(t.T(), rc, t.rr.wrapped.reader)
	require.Equal(t.T(), requestSize, data.Size)
	require.Equal(t.T(), "abcdef", string(buf[:data.Size]))
	assert.Equal(t.T(), uint64(requestSize), t.rr.wrapped.totalReadBytes.Load())
	assert.Equal(t.T(), int64(2+requestSize), t.rr.wrapped.expectedOffset.Load())
	assert.Equal(t.T(), expectedHandleInRequest, t.rr.wrapped.readHandle)
}

func (t *RandomReaderStretchrTest) Test_ReadAt_ExistingReaderLimitIsLessThanRequestedObjectSize() {
	t.object.Size = 5
	// Simulate an existing reader
	t.rr.wrapped.reader = &fake.FakeReader{ReadCloser: getReadCloser([]byte("xxx")), Handle: []byte("fake")}
	t.rr.wrapped.cancel = func() {}
	t.rr.wrapped.start = 0
	t.rr.wrapped.limit = 3
	rc := &fake.FakeReader{ReadCloser: getReadCloser([]byte("abcde"))}
	expectedHandleInRequest := t.rr.wrapped.reader.ReadHandle()
	readObjectRequest := &gcs.ReadObjectRequest{
		Name:       t.rr.wrapped.object.Name,
		Generation: t.rr.wrapped.object.Generation,
		Range: &gcs.ByteRange{
			Start: uint64(0),
			Limit: t.object.Size,
		},
		ReadCompressed: t.rr.wrapped.object.HasContentEncodingGzip(),
		ReadHandle:     expectedHandleInRequest,
	}
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, readObjectRequest).Return(rc, nil)
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{}).Times(2)
	requestSize := 6
	buf := make([]byte, requestSize)

	data, err := t.rr.ReadAt(buf, 0)

	require.Nil(t.T(), err)
	require.Nil(t.T(), t.rr.wrapped.reader)
	require.Equal(t.T(), int(t.object.Size), data.Size)
	require.Equal(t.T(), "abcde", string(buf[:data.Size]))
	assert.Equal(t.T(), t.object.Size, t.rr.wrapped.totalReadBytes.Load())
	assert.Equal(t.T(), int64(t.object.Size), t.rr.wrapped.expectedOffset.Load())
	assert.Equal(t.T(), []byte(nil), t.rr.wrapped.readHandle)
}

func (t *RandomReaderStretchrTest) Test_ReadAt_InvalidOffset() {
	testCases := []struct {
		name     string
		dataSize int
		start    int
	}{
		{
			name:     "InvalidOffset",
			dataSize: 50,
			start:    68,
		},
		{
			name:     "NegativeOffset",
			dataSize: 100,
			start:    -50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rr.wrapped.reader = nil
			t.object.Size = uint64(tc.dataSize)
			buf := make([]byte, tc.dataSize)

			_, err := t.rr.wrapped.ReadAt(t.rr.ctx, buf, int64(tc.start))

			assert.Error(t.T(), err)
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadAt_ValidateReadType() {
	testCases := []struct {
		name              string
		dataSize          int
		bucketType        gcs.BucketType
		readRanges        [][]int
		expectedReadTypes []int64
		expectedSeeks     []int
	}{
		{
			name:              "SequentialReadFlat",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: false},
			readRanges:        [][]int{{0, 10}, {10, 20}, {20, 35}, {35, 50}},
			expectedReadTypes: []int64{metrics.ReadTypeSequential, metrics.ReadTypeSequential, metrics.ReadTypeSequential, metrics.ReadTypeSequential},
			expectedSeeks:     []int{0, 0, 0, 0, 0},
		},
		{
			name:              "SequentialReadZonal",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: true},
			readRanges:        [][]int{{0, 10}, {10, 20}, {20, 35}, {35, 50}},
			expectedReadTypes: []int64{metrics.ReadTypeSequential, metrics.ReadTypeSequential, metrics.ReadTypeSequential, metrics.ReadTypeSequential},
			expectedSeeks:     []int{0, 0, 0, 0, 0},
		},
		{
			name:              "RandomReadFlat",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: false},
			readRanges:        [][]int{{0, 50}, {30, 40}, {10, 20}, {20, 30}, {30, 40}},
			expectedReadTypes: []int64{metrics.ReadTypeSequential, metrics.ReadTypeSequential, metrics.ReadTypeRandom, metrics.ReadTypeRandom, metrics.ReadTypeRandom},
			expectedSeeks:     []int{0, 1, 2, 2, 2},
		},
		{
			name:              "RandomReadZonal",
			dataSize:          100,
			bucketType:        gcs.BucketType{Zonal: true},
			readRanges:        [][]int{{0, 50}, {30, 40}, {10, 20}, {20, 30}, {30, 40}},
			expectedReadTypes: []int64{metrics.ReadTypeSequential, metrics.ReadTypeSequential, metrics.ReadTypeRandom, metrics.ReadTypeRandom, metrics.ReadTypeRandom},
			expectedSeeks:     []int{0, 1, 2, 2, 2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			assert.Equal(t.T(), len(tc.readRanges), len(tc.expectedReadTypes), "Test Parameter Error: readRanges and expectedReadTypes should have same length")
			t.rr.wrapped.reader = nil
			t.rr.wrapped.isMRDInUse.Store(false)
			t.rr.wrapped.seeks.Store(0)
			t.rr.wrapped.readType.Store(metrics.ReadTypeSequential)
			t.rr.wrapped.expectedOffset.Store(0)
			t.object.Size = uint64(tc.dataSize)
			testContent := testutil.GenerateRandomBytes(int(t.object.Size))
			fakeMRDWrapper, err := NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{}, &cfg.Config{})
			assert.Nil(t.T(), err, "Error in creating MRDWrapper")
			t.rr.wrapped.mrdWrapper = &fakeMRDWrapper
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond))
			t.mockBucket.On("BucketType", mock.Anything).Return(tc.bucketType).Times(len(tc.readRanges) * 2)

			for i, readRange := range tc.readRanges {
				t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(&fake.FakeReader{ReadCloser: getReadCloser(testContent)}, nil).Once()
				buf := make([]byte, readRange[1]-readRange[0])

				_, err := t.rr.wrapped.ReadAt(t.rr.ctx, buf, int64(readRange[0]))

				assert.NoError(t.T(), err)
				assert.Equal(t.T(), tc.expectedReadTypes[i], t.rr.wrapped.readType.Load())
				assert.Equal(t.T(), int64(readRange[1]), t.rr.wrapped.expectedOffset.Load())
				assert.Equal(t.T(), uint64(tc.expectedSeeks[i]), t.rr.wrapped.seeks.Load())
			}
		})
	}
}

// This test validates the bug fix where seeks are not updated correctly in case of zonal bucket random reads (b/410904634).
func (t *RandomReaderStretchrTest) Test_ReadAt_ValidateZonalRandomReads() {
	t.rr.wrapped.reader = nil
	t.rr.wrapped.isMRDInUse.Store(false)
	t.rr.wrapped.seeks.Store(0)
	t.rr.wrapped.readType.Store(metrics.ReadTypeSequential)
	t.rr.wrapped.expectedOffset.Store(0)
	t.rr.wrapped.totalReadBytes.Store(0)
	t.object.Size = 20 * MiB
	t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true})
	testContent := testutil.GenerateRandomBytes(int(t.object.Size))
	fakeMRDWrapper, err := NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{}, &cfg.Config{})
	assert.Nil(t.T(), err, "Error in creating MRDWrapper")
	t.rr.wrapped.mrdWrapper = &fakeMRDWrapper
	t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, mock.Anything).Return(&fake.FakeReader{ReadCloser: getReadCloser(testContent)}, nil).Twice()
	buf := make([]byte, 3*MiB)

	// Sequential read #1
	_, err = t.rr.wrapped.ReadAt(t.rr.ctx, buf, 13*MiB)
	assert.NoError(t.T(), err)
	// Random read #1
	seeks := 1
	_, err = t.rr.wrapped.ReadAt(t.rr.ctx, buf, 12*MiB)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), uint64(seeks), t.rr.wrapped.seeks.Load())

	readRanges := [][]int{{11 * MiB, 15 * MiB}, {12 * MiB, 14 * MiB}, {10 * MiB, 12 * MiB}, {9 * MiB, 11 * MiB}, {8 * MiB, 10 * MiB}}
	// Series of random reads to check if seeks are updated correctly and MRD is invoked always
	for _, readRange := range readRanges {
		seeks++
		t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond))
		buf := make([]byte, readRange[1]-readRange[0])

		_, err := t.rr.wrapped.ReadAt(t.rr.ctx, buf, int64(readRange[0]))

		assert.NoError(t.T(), err)
		assert.Equal(t.T(), metrics.ReadTypeRandom, t.rr.wrapped.readType.Load())
		assert.Equal(t.T(), int64(readRange[1]), t.rr.wrapped.expectedOffset.Load())
		assert.Equal(t.T(), uint64(seeks), t.rr.wrapped.seeks.Load())
	}
}

func (t *RandomReaderStretchrTest) Test_ReadAt_MRDRead() {
	testCases := []struct {
		name        string
		dataSize    int
		offset      int
		bytesToRead int
	}{
		{
			name:        "ReadChunk",
			dataSize:    100,
			offset:      37,
			bytesToRead: 43,
		},
		{
			name:        "ReadZeroByte",
			dataSize:    100,
			offset:      37,
			bytesToRead: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.rr.wrapped.reader = nil
			t.rr.wrapped.isMRDInUse.Store(false)
			t.rr.wrapped.expectedOffset.Store(10)
			t.rr.wrapped.seeks.Store(minSeeksForRandom + 1)
			t.object.Size = uint64(tc.dataSize)
			testContent := testutil.GenerateRandomBytes(int(t.object.Size))
			fakeMRDWrapper, err := NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{}, &cfg.Config{})
			assert.Nil(t.T(), err, "Error in creating MRDWrapper")
			t.rr.wrapped.mrdWrapper = &fakeMRDWrapper
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond)).Times(1)
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Times(1)
			buf := make([]byte, tc.bytesToRead)

			objData, err := t.rr.wrapped.ReadAt(t.rr.ctx, buf, int64(tc.offset))

			t.mockBucket.AssertNotCalled(t.T(), "NewReaderWithReadHandle", mock.Anything)
			assert.NoError(t.T(), err)
			assert.Nil(t.T(), t.rr.wrapped.reader)
			assert.Equal(t.T(), tc.bytesToRead, objData.Size)
			assert.Equal(t.T(), testContent[tc.offset:tc.offset+tc.bytesToRead], objData.DataBuf[:objData.Size])
			if tc.bytesToRead != 0 {
				assert.Equal(t.T(), int64(tc.offset+tc.bytesToRead), t.rr.wrapped.expectedOffset.Load())
			}
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromMultiRangeReader_ReadFull() {
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
			t.rr.wrapped.reader = nil
			t.rr.wrapped.isMRDInUse.Store(false)
			t.object.Size = uint64(tc.dataSize)
			testContent := testutil.GenerateRandomBytes(int(t.object.Size))
			fakeMRDWrapper, err := NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{}, &cfg.Config{})
			assert.Nil(t.T(), err, "Error in creating MRDWrapper")
			t.rr.wrapped.mrdWrapper = &fakeMRDWrapper
			t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond)).Times(1)
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Times(1)
			buf := make([]byte, tc.dataSize+tc.extraSize)

			bytesRead, err := t.rr.wrapped.readFromMultiRangeReader(t.rr.ctx, buf, 0, int64(t.object.Size), TestTimeoutForMultiRangeRead)

			assert.NoError(t.T(), err)
			assert.Equal(t.T(), tc.dataSize, bytesRead)
			assert.Equal(t.T(), testContent[:tc.dataSize], buf[:bytesRead])
			assert.Equal(t.T(), int64(t.object.Size), t.rr.wrapped.expectedOffset.Load())
		})
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromMultiRangeReader_ReadChunk() {
	testCases := []struct {
		name     string
		dataSize int
		start    int
		end      int
	}{
		{
			name:     "ReadChunk",
			dataSize: 100,
			start:    37,
			end:      93,
		},
	}

	for _, tc := range testCases {
		t.rr.wrapped.reader = nil
		t.object.Size = uint64(tc.dataSize)
		testContent := testutil.GenerateRandomBytes(int(t.object.Size))
		fakeMRDWrapper, err := NewMultiRangeDownloaderWrapperWithClock(t.mockBucket, t.object, &clock.FakeClock{}, &cfg.Config{})
		assert.Nil(t.T(), err, "Error in creating MRDWrapper")
		t.rr.wrapped.mrdWrapper = &fakeMRDWrapper
		t.mockBucket.On("NewMultiRangeDownloader", mock.Anything, mock.Anything).Return(fake.NewFakeMultiRangeDownloaderWithSleep(t.object, testContent, time.Microsecond)).Times(1)
		t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: true}).Times(1)
		buf := make([]byte, tc.end-tc.start)

		bytesRead, err := t.rr.wrapped.readFromMultiRangeReader(t.rr.ctx, buf, int64(tc.start), int64(tc.end), TestTimeoutForMultiRangeRead)

		assert.NoError(t.T(), err)
		assert.Equal(t.T(), tc.end-tc.start, bytesRead)
		assert.Equal(t.T(), testContent[tc.start:tc.end], buf[:bytesRead])
		assert.Equal(t.T(), int64(tc.end), t.rr.wrapped.expectedOffset.Load())
	}
}

func (t *RandomReaderStretchrTest) Test_ReadFromMultiRangeReader_NilMRDWrapper() {
	t.rr.wrapped.mrdWrapper = nil

	bytesRead, err := t.rr.wrapped.readFromMultiRangeReader(t.rr.ctx, make([]byte, t.object.Size), 0, int64(t.object.Size), TestTimeoutForMultiRangeRead)

	assert.ErrorContains(t.T(), err, "readFromMultiRangeReader: Invalid MultiRangeDownloaderWrapper")
	assert.Equal(t.T(), 0, bytesRead)
}

// Validates:
// 1. No change in ReadAt behavior based inactiveStreamTimeout config.
// 2. Valid timeout config creates InactiveTimeoutReader instance of storage.Reader.
func (t *RandomReaderStretchrTest) Test_ReadAt_WithAndWithoutReadConfig() {
	testCases := []struct {
		name                        string
		config                      *cfg.Config
		expectInactiveTimeoutReader bool
	}{
		{
			name:                        "WithoutReadConfig",
			config:                      nil,
			expectInactiveTimeoutReader: false,
		},
		{
			name:                        "WithReadConfigAndZeroTimeout",
			config:                      &cfg.Config{Read: cfg.ReadConfig{InactiveStreamTimeout: 0}},
			expectInactiveTimeoutReader: false,
		},
		{
			name:                        "WithReadConfigAndPositiveTimeout",
			config:                      &cfg.Config{Read: cfg.ReadConfig{InactiveStreamTimeout: 10 * time.Millisecond}},
			expectInactiveTimeoutReader: true,
		},
	}

	objectSize := uint64(20)
	readOffset := int64(0)
	readLength := 10 // Reading only 10 bytes from the complete object reader.

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			t.SetupTest() // Resets mockBucket, rr, etc. for each sub-test
			defer t.TearDownTest()

			t.rr.wrapped.config = tc.config
			t.rr.wrapped.reader = nil // Ensure startRead path is taken in ReadAt
			t.object.Size = objectSize
			// Prepare fake content for the GCS object.
			// startRead will attempt to read the entire object [0, objectSize)
			// because objectSize is small compared to typical sequentialReadSizeMb.
			fakeReaderContent := testutil.GenerateRandomBytes(int(t.object.Size))
			rc := &fake.FakeReader{ReadCloser: getReadCloser(fakeReaderContent)}
			expectedReadObjectRequest := &gcs.ReadObjectRequest{
				Name:       t.rr.wrapped.object.Name,
				Generation: t.rr.wrapped.object.Generation,
				Range: &gcs.ByteRange{
					Start: uint64(readOffset),    // Read from the beginning
					Limit: uint64(t.object.Size), // getReadInfo will determine this limit
				},
				ReadCompressed: t.rr.wrapped.object.HasContentEncodingGzip(),
				ReadHandle:     nil, // No existing read handle
			}
			t.mockBucket.On("NewReaderWithReadHandle", mock.Anything, expectedReadObjectRequest).Return(rc, nil).Once()
			// BucketType is called by ReadAt -> getReadInfo -> readerType to determine reader strategy.
			t.mockBucket.On("BucketType", mock.Anything).Return(gcs.BucketType{Zonal: false}).Twice()
			buf := make([]byte, readLength)

			objectData, err := t.rr.ReadAt(buf, readOffset)

			t.mockBucket.AssertExpectations(t.T())
			assert.NoError(t.T(), err)
			assert.Equal(t.T(), readLength, objectData.Size)
			assert.Equal(t.T(), fakeReaderContent[:readLength], buf[:objectData.Size]) // Ensure buffer is populated correctly
			assert.NotNil(t.T(), t.rr.wrapped.reader, "Reader should be active as partial data read from the requested range.")
			assert.NotNil(t.T(), t.rr.wrapped.cancel)
			assert.Equal(t.T(), int64(readLength), t.rr.wrapped.start)
			assert.Equal(t.T(), int64(t.object.Size), t.rr.wrapped.limit)
			_, isInactiveTimeoutReader := t.rr.wrapped.reader.(*InactiveTimeoutReader)
			assert.Equal(t.T(), tc.expectInactiveTimeoutReader, isInactiveTimeoutReader)
		})
	}
}
