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

package downloader

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vipnydav/gcsfuse/v3/internal/cache/data"
	"github.com/vipnydav/gcsfuse/v3/internal/cache/lru"
	"github.com/vipnydav/gcsfuse/v3/internal/storage"
	"github.com/vipnydav/gcsfuse/v3/internal/storage/gcs"
)

func getMinObject(objectName string, bucket gcs.Bucket) gcs.MinObject {
	ctx := context.Background()
	minObject, _, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: objectName,
		ForceFetchFromGcs: true})
	if err != nil {
		panic(fmt.Errorf("error occured while statting the object: %w", err))
	}
	if minObject != nil {
		return *minObject
	}
	return gcs.MinObject{}
}

func verifyFileTillOffset(t *testing.T, spec data.FileSpec, offset int64, content []byte) {
	fileStat, err := os.Stat(spec.Path)
	if assert.Nil(t, err) {
		// Verify the content of file downloaded only till the size of content passed.
		fileContent, err := os.ReadFile(spec.Path)
		assert.Nil(t, err)
		assert.Equal(t, spec.FilePerm, fileStat.Mode())
		assert.GreaterOrEqual(t, len(content), int(offset))
		assert.GreaterOrEqual(t, len(fileContent), int(offset))
		assert.Equal(t, content[:offset], fileContent[:offset])
	}

}

func verifyCompleteFile(t *testing.T, spec data.FileSpec, content []byte) {
	fileStat, err := os.Stat(spec.Path)
	assert.Equal(t, nil, err)
	assert.Equal(t, spec.FilePerm, fileStat.Mode())
	assert.LessOrEqual(t, int64(len(content)), fileStat.Size())
	// Verify the content of file downloaded only till the size of content passed.
	fileContent, err := os.ReadFile(spec.Path)
	assert.Equal(t, nil, err)
	assert.True(t, reflect.DeepEqual(content, fileContent[:len(content)]))
}

func verifyFileInfoEntry(t *testing.T, mockBucket *storage.TestifyMockBucket, object gcs.MinObject, cache *lru.Cache, offset uint64) {
	fileInfo := getFileInfo(t, mockBucket, object, cache)
	assert.True(t, fileInfo != nil)
	assert.Equal(t, object.Generation, fileInfo.(data.FileInfo).ObjectGeneration)
	assert.LessOrEqual(t, offset, fileInfo.(data.FileInfo).Offset)
	assert.Equal(t, object.Size, fileInfo.(data.FileInfo).Size())
}

func getFileInfo(t *testing.T, mockBucket *storage.TestifyMockBucket, object gcs.MinObject, cache *lru.Cache) lru.ValueType {
	fileInfoKey := data.FileInfoKey{BucketName: mockBucket.Name(), ObjectName: object.Name}
	fileInfoKeyName, err := fileInfoKey.Key()
	assert.Equal(t, nil, err)
	return cache.LookUp(fileInfoKeyName)
}
