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

package downloader

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/config"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	"github.com/stretchr/testify/assert"
)

func getMinObject(t *testing.T, objectName string, bucket gcs.Bucket) gcs.MinObject {
	t.Helper()
	ctx := context.Background()
	minObject, _, err := bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: objectName,
		ForceFetchFromGcs: true})
	if err != nil {
		panic(fmt.Errorf("error whlie stating object: %w", err))
	}
	if minObject != nil {
		return *minObject
	}
	return gcs.MinObject{}
}

func CreateObjectInStoreAndInitCache(t *testing.T, objectSize int64) (gcs.MinObject, gcs.Bucket, *lru.Cache) {
	t.Helper()
	fakeStorage := storage.NewFakeStorage()
	t.Cleanup(func() { fakeStorage.ShutDown() })
	storageHandle := fakeStorage.CreateStorageHandle()
	bucket := storageHandle.BucketHandle(storage.TestBucketName, "")
	objectName := "path/in/gcs/foo.txt"
	objectContent := make([]byte, objectSize)
	_, err := rand.Read(objectContent)
	if err != nil {
		t.Fatalf("Error while generating random object content: %v", err)
	}
	_, err = storageutil.CreateObject(context.Background(), bucket, objectName, objectContent)
	if err != nil {
		t.Fatalf("Error while creating object in fakestorage: %v", err)
	}
	minObj := getMinObject(t, objectName, bucket)
	cacheDir, err := os.MkdirTemp("", "gcsfuse_test")
	if err != nil {
		t.Fatalf("Error while creating the cache directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(cacheDir) })
	cache := lru.NewCache(uint64(2 * objectSize))
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: objectName,
	}
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: minObj.Generation,
		FileSize:         minObj.Size,
		Offset:           0,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		t.Fatalf("Error occurred while retrieving fileInfoKey: %v", err)
	}
	_, err = cache.Insert(fileInfoKeyName, fileInfo)
	if err != nil {
		t.Fatalf("Error occurred while inserting fileinfo into cache: %v", err)
	}
	return minObj, bucket, cache
}

func TestParallelDownloads(t *testing.T) {
	tbl := []struct {
		name                   string
		objectSize             int64
		readReqSize            int
		maxDownloadParallelism int
		downloadOffset         int64
		expectedOffset         int64
		subscribedOffset       int64
	}{
		{
			name:                   "download in chunks of concurrency * readReqSize",
			objectSize:             15 * util.MiB,
			readReqSize:            4,
			maxDownloadParallelism: 3,
			subscribedOffset:       7,
			downloadOffset:         10,
			expectedOffset:         12 * util.MiB,
		},
		{
			name:                   "download only upto the object size",
			objectSize:             10 * util.MiB,
			readReqSize:            4,
			maxDownloadParallelism: 3,
			subscribedOffset:       7,
			downloadOffset:         10,
			expectedOffset:         10 * util.MiB,
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			minObj, bucket, cache := CreateObjectInStoreAndInitCache(t, tc.objectSize)
			jm := NewJobManager(cache, util.DefaultFilePerm, util.DefaultDirPerm, cacheDir, 2, &config.FileCacheConfig{EnableParallelDownloads: true,
				DownloadParallelismPerFile: math.MaxInt, ReadRequestSizeMB: tc.readReqSize, EnableCrcCheck: true, MaxDownloadParallelism: tc.maxDownloadParallelism})
			job := jm.CreateJobIfNotExists(&minObj, bucket)
			subscriberC := job.subscribe(tc.subscribedOffset)

			job.Download(context.Background(), 10, false)

			for {
				select {
				case jobStatus := <-subscriberC:
					assert.Equal(t, tc.expectedOffset, jobStatus.Offset)
					return
				case <-time.After(1 * time.Second):
					assert.Fail(t, "Test timed out")
					return
				}
			}
		})
	}
}
