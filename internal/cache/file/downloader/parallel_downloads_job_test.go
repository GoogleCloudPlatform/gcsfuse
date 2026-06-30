// Copyright 2024 Google LLC
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
//
// File that contains tests specific to parallel download job i.e. when
// EnableParallelDownloads=true.

package downloader

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	testutil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type parallelDownloaderTest struct {
	downloaderTest
}

func newParallelDownloaderTest(t *testing.T) *parallelDownloaderTest {
	dt := &parallelDownloaderTest{
		downloaderTest: downloaderTest{
			t:       t,
			assert:  assert.New(t),
			require: require.New(t),
		},
	}
	dt.defaultFileCacheConfig = &cfg.FileCacheConfig{
		ExperimentalParallelDownloadsDefaultOn: true,
		EnableParallelDownloads:                true,
		ParallelDownloadsPerFile:               3,
		DownloadChunkSizeMb:                    3,
		EnableCrc:                              true,
		WriteBufferSize:                        4 * 1024 * 1024,
	}
	dt.setupHelper()
	return dt
}

func TestParallel_downloadRange(t *testing.T) {
	dt := newParallelDownloaderTest(t)
	defer dt.tearDown()

	// Create object in fake GCS
	objectName := "path/in/gcs/foo.txt"
	objectSize := 10 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecuted atomic.Bool
	removeCallback := func() { callbackExecuted.Store(true) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), removeCallback)
	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())
	defer dt.job.cancelFunc()
	file, err := util.CreateFile(data.FileSpec{Path: dt.job.fileSpec.Path,
		FilePerm: os.FileMode(0600), DirPerm: os.FileMode(0700)}, os.O_TRUNC|os.O_RDWR)
	dt.require.NoError(err)
	verifyContentAtOffset := func(file *os.File, start, end int64) {
		_, err = file.Seek(int64(start), 0)
		dt.require.NoError(err)
		buf := make([]byte, end-start)
		_, err = file.Read(buf)
		dt.require.NoError(err)
		// If content don't match then print start and end for easy debuggability.
		dt.require.Equal(objectContent[start:end], buf, fmt.Sprintf("content didn't match for start: %v and end: %v", start, end))
	}

	// Download end 1MiB of object
	start, end := int64(9*util.MiB), int64(10*util.MiB)
	offsetWriter := io.NewOffsetWriter(file, start)
	rangeMap := make(map[int64]int64)

	_, err = dt.job.downloadRange(context.Background(), offsetWriter, start, end, nil, rangeMap)
	dt.require.NoError(err)
	verifyContentAtOffset(file, start, end)

	// Download start 4MiB of object
	start, end = int64(0*util.MiB), int64(4*util.MiB)
	offsetWriter = io.NewOffsetWriter(file, start)
	_, err = dt.job.downloadRange(context.Background(), offsetWriter, start, end, nil, rangeMap)
	dt.require.NoError(err)
	verifyContentAtOffset(file, start, end)
	dt.require.Equal(int64(4*util.MiB), rangeMap[start])

	// Download middle 1B of object
	start, end = int64(5*util.MiB), int64(5*util.MiB+1)
	offsetWriter = io.NewOffsetWriter(file, start)
	_, err = dt.job.downloadRange(context.Background(), offsetWriter, start, end, nil, rangeMap)
	dt.require.NoError(err)
	verifyContentAtOffset(file, start, end)
	dt.require.Equal(int64(5*util.MiB+1), rangeMap[start])

	// Download 0B of object
	start, end = int64(5*util.MiB), int64(5*util.MiB)
	offsetWriter = io.NewOffsetWriter(file, start)
	_, err = dt.job.downloadRange(context.Background(), offsetWriter, start, end, nil, rangeMap)
	dt.require.NoError(err)
	verifyContentAtOffset(file, start, end)
	dt.require.Equal(int64(5*util.MiB+1), rangeMap[start])
}

func TestParallel_parallelDownloadObjectToFile(t *testing.T) {
	dt := newParallelDownloaderTest(t)
	defer dt.tearDown()

	objectName := "path/in/gcs/foo.txt"
	objectSize := 10 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})
	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())
	defer dt.job.cancelFunc()
	// Add subscriber
	subscribedOffset := int64(1 * util.MiB)
	notificationC := dt.job.subscribe(subscribedOffset)
	file, err := util.CreateFile(data.FileSpec{Path: dt.job.fileSpec.Path,
		FilePerm: os.FileMode(0600), DirPerm: os.FileMode(0700)}, os.O_TRUNC|os.O_RDWR)
	dt.require.NoError(err)
	defer func() {
		_ = file.Close()
	}()

	// Start download
	err = dt.job.parallelDownloadObjectToFile(file)

	dt.require.NoError(err)
	jobStatus, ok := <-notificationC
	dt.require.True(ok)
	// Check the notification is sent after subscribed offset
	dt.require.GreaterOrEqual(jobStatus.Offset, subscribedOffset)
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	// Verify file is downloaded
	dt.verifyFile(objectContent)
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(objectSize))
}

func TestParallel_parallelDownloadObjectToFile_CtxCancelled(t *testing.T) {
	dt := newParallelDownloaderTest(t)
	defer dt.tearDown()

	objectName := "path/in/gcs/cancel.txt"
	objectSize := util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2), func() {})
	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())
	file, err := util.CreateFile(data.FileSpec{Path: dt.job.fileSpec.Path,
		FilePerm: os.FileMode(0600), DirPerm: os.FileMode(0700)}, os.O_TRUNC|os.O_RDWR)
	dt.require.NoError(err)
	defer func() {
		_ = file.Close()
	}()

	dt.job.cancelFunc()
	err = dt.job.parallelDownloadObjectToFile(file)

	dt.require.ErrorIs(err, context.Canceled)
}

func TestParallel_updateRangeMap_withNoEntries(t *testing.T) {
	dt := newParallelDownloaderTest(t)
	defer dt.tearDown()

	rangeMap := make(map[int64]int64)

	err := dt.job.updateRangeMap(rangeMap, 0, 10)

	dt.require.NoError(err)
	dt.require.Len(rangeMap, 2)
	dt.require.Equal(int64(10), rangeMap[0])
	dt.require.Equal(int64(0), rangeMap[10])
}

func TestParallel_updateRangeMap_withInputContinuousWithEndOffset(t *testing.T) {
	dt := newParallelDownloaderTest(t)
	defer dt.tearDown()

	rangeMap := make(map[int64]int64)
	rangeMap[0] = 2
	rangeMap[2] = 0
	rangeMap[4] = 6
	rangeMap[6] = 4

	err := dt.job.updateRangeMap(rangeMap, 6, 8)

	dt.require.NoError(err)
	dt.require.Len(rangeMap, 4)
	dt.require.Equal(int64(2), rangeMap[0])
	dt.require.Equal(int64(0), rangeMap[2])
	dt.require.Equal(int64(8), rangeMap[4])
	dt.require.Equal(int64(4), rangeMap[8])
}

func TestParallel_updateRangeMap_withInputContinuousWithStartOffset(t *testing.T) {
	dt := newParallelDownloaderTest(t)
	defer dt.tearDown()

	rangeMap := make(map[int64]int64)
	rangeMap[2] = 4
	rangeMap[4] = 2
	rangeMap[8] = 10
	rangeMap[10] = 8

	err := dt.job.updateRangeMap(rangeMap, 0, 2)

	dt.require.NoError(err)
	dt.require.Len(rangeMap, 4)
	dt.require.Equal(int64(4), rangeMap[0])
	dt.require.Equal(int64(0), rangeMap[4])
	dt.require.Equal(int64(10), rangeMap[8])
	dt.require.Equal(int64(8), rangeMap[10])
}

func TestParallel_updateRangeMap_withInputFillingTheMissingRange(t *testing.T) {
	dt := newParallelDownloaderTest(t)
	defer dt.tearDown()

	rangeMap := make(map[int64]int64)
	rangeMap[0] = 4
	rangeMap[4] = 0
	rangeMap[6] = 8
	rangeMap[8] = 6

	err := dt.job.updateRangeMap(rangeMap, 4, 6)

	dt.require.NoError(err)
	dt.require.Len(rangeMap, 2)
	dt.require.Equal(int64(8), rangeMap[0])
	dt.require.Equal(int64(0), rangeMap[8])
}

func TestParallel_updateRangeMap_withInputNotOverlappingWithAnyRanges(t *testing.T) {
	dt := newParallelDownloaderTest(t)
	defer dt.tearDown()

	rangeMap := make(map[int64]int64)
	rangeMap[0] = 4
	rangeMap[4] = 0
	rangeMap[12] = 14
	rangeMap[14] = 12

	err := dt.job.updateRangeMap(rangeMap, 8, 10)

	dt.require.NoError(err)
	dt.require.Len(rangeMap, 6)
	dt.require.Equal(int64(0), rangeMap[4])
	dt.require.Equal(int64(4), rangeMap[0])
	dt.require.Equal(int64(8), rangeMap[10])
	dt.require.Equal(int64(10), rangeMap[8])
	dt.require.Equal(int64(12), rangeMap[14])
	dt.require.Equal(int64(14), rangeMap[12])
}
