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
	"os"
	"path"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file/downloader"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	TestObject                = "testObject"
	sequentialReadSizeInMb    = 22
	sequentialReadSizeInBytes = sequentialReadSizeInMb * MB
	CacheMaxSize              = 2 * sequentialReadSizeInMb * util.MiB
)

type FileCacheReaderTest struct {
	suite.Suite
	object             *gcs.MinObject
	mockBucket         *storage.TestifyMockBucket
	cacheDir           string
	jobManager         *downloader.JobManager
	mockCacheHandler   file.CacheHandlerInterface
	mockCacheHandle    file.CacheHandleInterface
	mockMetricsHandler common.MetricHandle
}

func TestFileCacheReaderTestSuite(t *testing.T) {
	suite.Run(t, new(FileCacheReaderTest))
}

func (t *FileCacheReaderTest) SetupTestSuite() {
	t.object = &gcs.MinObject{
		Name:       TestObject,
		Size:       17,
		Generation: 1234,
	}
	t.mockBucket = new(storage.TestifyMockBucket)
	t.cacheDir = path.Join(os.Getenv("HOME"), "test_cache_dir")
	lruCache := lru.NewCache(CacheMaxSize)
	t.jobManager = downloader.NewJobManager(lruCache, util.DefaultFilePerm, util.DefaultDirPerm, t.cacheDir, sequentialReadSizeInMb, &cfg.FileCacheConfig{EnableCrc: false}, nil)
	t.mockCacheHandler = new(file.MockFileCacheHandler)
	t.mockCacheHandle = new(file.MockCacheHandle)
	t.mockMetricsHandler = new(common.MockMetricHandle)
}

func (t *FileCacheReaderTest) TearDownSuite() {
	err := os.RemoveAll(t.cacheDir)
	if err != nil {
		t.T().Logf("Failed to clean up test cache directory '%s': %v", t.cacheDir, err)
	}
}

func (t *FileCacheReaderTest) TestNewFileCacheReader() {
	reader, err := NewFileCacheReader(t.object, t.mockBucket, t.mockCacheHandler, true, nil, 0)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), reader)
	assert.Equal(t.T(), t.object, reader.obj)
	assert.Equal(t.T(), t.mockBucket, reader.bucket)
	assert.True(t.T(), reader.cacheFileForRangeRead)
	assert.Nil(t.T(), reader.metricHandle)
	assert.Nil(t.T(), reader.fileCacheHandle)
}
