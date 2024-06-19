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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	. "github.com/jacobsa/ogletest"
)

// TestParallelDownloader runs all the tests with parallel downloads job that
// are run without parallel downloads job as part of TestDownloader in
// downloader_test.go
func TestParallelDownloader(t *testing.T) { RunTests(t) }

type parallelDownloaderTest struct {
	downloaderTest
}

func init() {
	RegisterTestSuite(&parallelDownloaderTest{})
}

func (dt *parallelDownloaderTest) SetUp(*TestInfo) {
	dt.defaultFileCacheConfig = &config.FileCacheConfig{EnableParallelDownloads: true,
		DownloadParallelismPerFile: 5, ReadRequestSizeMB: 2, EnableCrcCheck: true, MaxDownloadParallelism: 6}
	dt.setupHelper()
}
