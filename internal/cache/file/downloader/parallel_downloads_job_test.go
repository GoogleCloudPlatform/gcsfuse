// Copyright 2024 Google Inc. All Rights Reserved.
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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	. "github.com/jacobsa/ogletest"
)

// TestParallelDownloader runs all the tests with parallel downloads job that
// are run without parallel downloads job as part of TestDownloader in
// downloader_test.go
func TestParallelDownloader(t *testing.T) { RunTests(t) }

type parallelDownloaderTest struct {
	downloaderTest
}

func init() { RegisterTestSuite(&parallelDownloaderTest{}) }

func (dt *parallelDownloaderTest) SetUp(*TestInfo) {
	dt.defaultFileCacheConfig = &config.FileCacheConfig{EnableParallelDownloads: true,
		DownloadParallelismPerFile: 3, ReadRequestSizeMB: 3, EnableCrcCheck: true}
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
	AssertEq(nil, err)
	verifyContentAtOffset := func(file *os.File, start, end int64) {
		_, err = file.Seek(start, 0)
		AssertEq(nil, err)
		buf := make([]byte, end-start)
		_, err = file.Read(buf)
		AssertEq(nil, err)
		// If content don't match then print start and end for easy debuggability.
		AssertTrue(reflect.DeepEqual(objectContent[start:end], buf), fmt.Sprintf("content didn't match for start: %v and end: %v", start, end))
	}

	// Download end 1MiB of object
	start, end := int64(9*util.MiB), int64(10*util.MiB)
	offsetWriter := io.NewOffsetWriter(file, start)
	err = dt.job.downloadRange(context.Background(), offsetWriter, start, end)
	AssertEq(nil, err)
	verifyContentAtOffset(file, start, end)

	// Download start 4MiB of object
	start, end = int64(0*util.MiB), int64(4*util.MiB)
	offsetWriter = io.NewOffsetWriter(file, start)
	err = dt.job.downloadRange(context.Background(), offsetWriter, start, end)
	AssertEq(nil, err)
	verifyContentAtOffset(file, start, end)

	// Download middle 1B of object
	start, end = int64(5*util.MiB), int64(5*util.MiB+1)
	offsetWriter = io.NewOffsetWriter(file, start)
	err = dt.job.downloadRange(context.Background(), offsetWriter, start, end)
	AssertEq(nil, err)
	verifyContentAtOffset(file, start, end)

	// Download 0B of object
	start, end = int64(5*util.MiB), int64(5*util.MiB)
	offsetWriter = io.NewOffsetWriter(file, start)
	err = dt.job.downloadRange(context.Background(), offsetWriter, start, end)
	AssertEq(nil, err)
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
	AssertEq(nil, err)
	defer func() {
		_ = file.Close()
	}()

	// Start download
	err = dt.job.parallelDownloadObjectToFile(file)

	AssertEq(nil, err)
	jobStatus, ok := <-notificationC
	AssertEq(true, ok)
	// Check the notification is sent after subscribed offset
	AssertGe(jobStatus.Offset, subscribedOffset)
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
	AssertEq(nil, err)
	defer func() {
		_ = file.Close()
	}()

	dt.job.cancelFunc()
	err = dt.job.parallelDownloadObjectToFile(file)

	AssertTrue(errors.Is(err, context.Canceled), fmt.Sprintf("didn't get context canceled error: %v", err))
}
