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
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type parallelDownloaderTest struct {
	downloaderTest
}

func TestParallelDownloaderTestSuite(t *testing.T) {
	suite.Run(t, new(parallelDownloaderTest))
}

func (dt *parallelDownloaderTest) SetupTest() {
	dt.defaultFileCacheConfig = &cfg.FileCacheConfig{
		EnableParallelDownloads:  true,
		ParallelDownloadsPerFile: 3,
		DownloadChunkSizeMb:      3,
		EnableCrc:                true,
		WriteBufferSize:          4 * 1024 * 1024,
	}
	dt.setupHelper()
}

func (dt *parallelDownloaderTest) Test_downloadRange() {
	// Create object in fake GCS
	objectName := "path/in/gcs/foo.txt"
	objectSize := 10 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecuted atomic.Bool
	removeCallback := func() { callbackExecuted.Store(true) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), removeCallback)
	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())
	file, err := util.CreateFile(data.FileSpec{Path: dt.job.fileSpec.Path,
		FilePerm: os.FileMode(0600), DirPerm: os.FileMode(0700)}, os.O_TRUNC|os.O_RDWR)
	assert.Equal(dt.T(), nil, err)
	verifyContentAtOffset := func(file *os.File, start, end int64) {
		_, err = file.Seek(int64(start), 0)
		assert.Equal(dt.T(), nil, err)
		buf := make([]byte, end-start)
		_, err = file.Read(buf)
		assert.Equal(dt.T(), nil, err)
		// If content don't match then print start and end for easy debuggability.
		assert.True(dt.T(), reflect.DeepEqual(objectContent[start:end], buf), fmt.Sprintf("content didn't match for start: %v and end: %v", start, end))
	}

	// Download end 1MiB of object
	start, end := int64(9*util.MiB), int64(10*util.MiB)
	offsetWriter := io.NewOffsetWriter(file, start)
	err = dt.job.downloadRange(context.Background(), offsetWriter, start, end)
	assert.Equal(dt.T(), nil, err)
	verifyContentAtOffset(file, start, end)

	// Download start 4MiB of object
	start, end = int64(0*util.MiB), int64(4*util.MiB)
	offsetWriter = io.NewOffsetWriter(file, start)
	err = dt.job.downloadRange(context.Background(), offsetWriter, start, end)
	assert.Equal(dt.T(), nil, err)
	verifyContentAtOffset(file, start, end)

	// Download middle 1B of object
	start, end = int64(5*util.MiB), int64(5*util.MiB+1)
	offsetWriter = io.NewOffsetWriter(file, start)
	err = dt.job.downloadRange(context.Background(), offsetWriter, start, end)
	assert.Equal(dt.T(), nil, err)
	verifyContentAtOffset(file, start, end)

	// Download 0B of object
	start, end = int64(5*util.MiB), int64(5*util.MiB)
	offsetWriter = io.NewOffsetWriter(file, start)
	err = dt.job.downloadRange(context.Background(), offsetWriter, start, end)
	assert.Equal(dt.T(), nil, err)
	verifyContentAtOffset(file, start, end)
}

func (dt *parallelDownloaderTest) Test_parallelDownloadObjectToFile() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 10 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})
	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())
	// Add subscriber
	subscribedOffset := int64(1 * util.MiB)
	notificationC := dt.job.subscribe(subscribedOffset)
	file, err := util.CreateFile(data.FileSpec{Path: dt.job.fileSpec.Path,
		FilePerm: os.FileMode(0600), DirPerm: os.FileMode(0700)}, os.O_TRUNC|os.O_RDWR)
	assert.Equal(dt.T(), nil, err)
	defer func() {
		_ = file.Close()
	}()

	// Start download
	err = dt.job.parallelDownloadObjectToFile(file)

	assert.Equal(dt.T(), nil, err)
	jobStatus, ok := <-notificationC
	assert.Equal(dt.T(), true, ok)
	// Check the notification is sent after subscribed offset
	assert.GreaterOrEqual(dt.T(), jobStatus.Offset, subscribedOffset)
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	// Verify file is downloaded
	dt.verifyFile(objectContent)
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(objectSize))
}

func (dt *parallelDownloaderTest) Test_parallelDownloadObjectToFile_CtxCancelled() {
	objectName := "path/in/gcs/cancel.txt"
	objectSize := util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2), func() {})
	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())
	file, err := util.CreateFile(data.FileSpec{Path: dt.job.fileSpec.Path,
		FilePerm: os.FileMode(0600), DirPerm: os.FileMode(0700)}, os.O_TRUNC|os.O_RDWR)
	assert.Equal(dt.T(), nil, err)
	defer func() {
		_ = file.Close()
	}()

	dt.job.cancelFunc()
	err = dt.job.parallelDownloadObjectToFile(file)

	assert.True(dt.T(), errors.Is(err, context.Canceled), fmt.Sprintf("didn't get context canceled error: %v", err))
}

func (dt *parallelDownloaderTest) Test_updateRangeMap_withNoEntries() {
	rangeMap := make(map[int64]int64)

	err := dt.job.updateRangeMap(rangeMap, 0, 10)

	assert.Equal(dt.T(), nil, err)
	assert.EqualValues(dt.T(), 2, len(rangeMap))
	assert.EqualValues(dt.T(), 10, rangeMap[0])
	assert.EqualValues(dt.T(), 0, rangeMap[10])
}

func (dt *parallelDownloaderTest) Test_updateRangeMap_withInputContinuousWithEndOffset() {
	rangeMap := make(map[int64]int64)
	rangeMap[0] = 2
	rangeMap[2] = 0
	rangeMap[4] = 6
	rangeMap[6] = 4

	err := dt.job.updateRangeMap(rangeMap, 6, 8)

	assert.Equal(dt.T(), nil, err)
	assert.EqualValues(dt.T(), 4, len(rangeMap))
	assert.EqualValues(dt.T(), 2, rangeMap[0])
	assert.EqualValues(dt.T(), 0, rangeMap[2])
	assert.EqualValues(dt.T(), 8, rangeMap[4])
	assert.EqualValues(dt.T(), 4, rangeMap[8])
}

func (dt *parallelDownloaderTest) Test_updateRangeMap_withInputContinuousWithStartOffset() {
	rangeMap := make(map[int64]int64)
	rangeMap[2] = 4
	rangeMap[4] = 2
	rangeMap[8] = 10
	rangeMap[10] = 8

	err := dt.job.updateRangeMap(rangeMap, 0, 2)

	assert.Equal(dt.T(), nil, err)
	assert.EqualValues(dt.T(), 4, len(rangeMap))
	assert.EqualValues(dt.T(), 4, rangeMap[0])
	assert.EqualValues(dt.T(), 0, rangeMap[4])
	assert.EqualValues(dt.T(), 10, rangeMap[8])
	assert.EqualValues(dt.T(), 8, rangeMap[10])
}

func (dt *parallelDownloaderTest) Test_updateRangeMap_withInputFillingTheMissingRange() {
	rangeMap := make(map[int64]int64)
	rangeMap[0] = 4
	rangeMap[4] = 0
	rangeMap[6] = 8
	rangeMap[8] = 6

	err := dt.job.updateRangeMap(rangeMap, 4, 6)

	assert.Equal(dt.T(), nil, err)
	assert.EqualValues(dt.T(), 2, len(rangeMap))
	assert.EqualValues(dt.T(), 8, rangeMap[0])
	assert.EqualValues(dt.T(), 0, rangeMap[8])
}

func (dt *parallelDownloaderTest) Test_updateRangeMap_withInputNotOverlappingWithAnyRanges() {
	rangeMap := make(map[int64]int64)
	rangeMap[0] = 4
	rangeMap[4] = 0
	rangeMap[12] = 14
	rangeMap[14] = 12

	err := dt.job.updateRangeMap(rangeMap, 8, 10)

	assert.Equal(dt.T(), nil, err)
	assert.EqualValues(dt.T(), 6, len(rangeMap))
	assert.EqualValues(dt.T(), 0, rangeMap[4])
	assert.EqualValues(dt.T(), 4, rangeMap[0])
	assert.EqualValues(dt.T(), 8, rangeMap[10])
	assert.EqualValues(dt.T(), 10, rangeMap[8])
	assert.EqualValues(dt.T(), 12, rangeMap[14])
	assert.EqualValues(dt.T(), 14, rangeMap[12])
}
