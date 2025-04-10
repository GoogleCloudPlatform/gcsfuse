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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/file"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/stretchr/testify/assert"
)

const TestObject = "testObject"

func TestNewFileCacheReader(t *testing.T) {
	obj := &gcs.MinObject{
		Name:            TestObject,
		Size:            10,
		Generation:      0,
		MetaGeneration:  -1,
		Updated:         time.Time{},
		Metadata:        nil,
		ContentEncoding: "",
		CRC32C:          nil,
	}
	var bucket gcs.Bucket
	cacheHandler := &file.CacheHandler{}
	var metricHandle common.MetricHandle

	reader := NewFileCacheReader(obj, bucket, cacheHandler, true, metricHandle)

	assert.Equal(t, obj, reader.obj)
	assert.Equal(t, bucket, reader.bucket)
	assert.Equal(t, cacheHandler, reader.fileCacheHandler)
	assert.True(t, reader.cacheFileForRangeRead)
	assert.Equal(t, metricHandle, reader.metricHandle)
	assert.Nil(t, reader.fileCacheHandle) // Initially nil
}
