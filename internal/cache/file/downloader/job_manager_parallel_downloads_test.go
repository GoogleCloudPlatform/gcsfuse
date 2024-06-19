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
		t.Fatalf("Error occured while stating the object: %v", err)
	}
	if minObject != nil {
		return *minObject
	}
	return gcs.MinObject{}
}

func createObjectInStore(t *testing.T, objPath string, objSize int64, bucket gcs.Bucket) {
	t.Helper()
	objectContent := make([]byte, objSize)
	_, err := rand.Read(objectContent)
	if err != nil {
		t.Fatalf("Error while generating random object content: %v", err)
	}
	_, err = storageutil.CreateObject(context.Background(), bucket, objPath, objectContent)
	if err != nil {
		t.Fatalf("Error while creating object in fakestorage: %v", err)
	}
}

func configureFakeStorage(t *testing.T) storage.StorageHandle {
	t.Helper()
	fakeStorage := storage.NewFakeStorage()
	t.Cleanup(func() { fakeStorage.ShutDown() })
	return fakeStorage.CreateStorageHandle()
}

func configureCache(t *testing.T, maxSize int64) (*lru.Cache, string) {
	t.Helper()
	cache := lru.NewCache(uint64(maxSize))
	cacheDir, err := os.MkdirTemp("", "gcsfuse_test")
	if err != nil {
		t.Fatalf("Error while creating the cache directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(cacheDir) })
	return cache, cacheDir
}

func createObjectInStoreAndInitCache(t *testing.T, cache *lru.Cache, bucket gcs.Bucket, objectName string, objectSize int64) gcs.MinObject {
	t.Helper()
	createObjectInStore(t, objectName, objectSize, bucket)
	minObj := getMinObject(t, objectName, bucket)
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
	return minObj
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
			cache, cacheDir := configureCache(t, 2*tc.objectSize)
			storageHandle := configureFakeStorage(t)
			bucket := storageHandle.BucketHandle(storage.TestBucketName, "")
			minObj := createObjectInStoreAndInitCache(t, cache, bucket, "path/in/gcs/foo.txt", tc.objectSize)
			jm := NewJobManager(cache, util.DefaultFilePerm, util.DefaultDirPerm, cacheDir, 2, &config.FileCacheConfig{EnableParallelDownloads: true,
				DownloadParallelismPerFile: math.MaxInt, ReadRequestSizeMB: tc.readReqSize, EnableCrcCheck: true, MaxDownloadParallelism: tc.maxDownloadParallelism})
			job := jm.CreateJobIfNotExists(&minObj, bucket)
			subscriberC := job.subscribe(tc.subscribedOffset)

			_, err := job.Download(context.Background(), 10, false)

			timeout := time.After(1 * time.Second)
			for {
				select {
				case jobStatus := <-subscriberC:
					if assert.Nil(t, err) {
						assert.Equal(t, tc.expectedOffset, jobStatus.Offset)
					}
					return
				case <-timeout:
					assert.Fail(t, "Test timed out")
					return
				}
			}
		})
	}
}

func TestMultipleConcurrentDownloads(t *testing.T) {
	t.Parallel()
	storageHandle := configureFakeStorage(t)
	cache, cacheDir := configureCache(t, 30*util.MiB)
	bucket := storageHandle.BucketHandle(storage.TestBucketName, "")
	minObj1 := createObjectInStoreAndInitCache(t, cache, bucket, "path/in/gcs/foo.txt", 10*util.MiB)
	minObj2 := createObjectInStoreAndInitCache(t, cache, bucket, "path/in/gcs/bar.txt", 5*util.MiB)
	jm := NewJobManager(cache, util.DefaultFilePerm, util.DefaultDirPerm, cacheDir, 2, &config.FileCacheConfig{EnableParallelDownloads: true,
		DownloadParallelismPerFile: math.MaxInt, ReadRequestSizeMB: 2, EnableCrcCheck: true, MaxDownloadParallelism: 2})
	job1 := jm.CreateJobIfNotExists(&minObj1, bucket)
	job2 := jm.CreateJobIfNotExists(&minObj2, bucket)
	s1 := job1.subscribe(10 * util.MiB)
	s2 := job2.subscribe(5 * util.MiB)
	ctx := context.Background()

	_, err1 := job1.Download(ctx, 10*util.MiB, false)
	_, err2 := job2.Download(ctx, 5*util.MiB, false)

	notif1, notif2 := false, false
	timeout := time.After(1000 * time.Second)
	for {
		select {
		case <-s1:
			notif1 = true
		case <-s2:
			notif2 = true
		case <-timeout:
			assert.Fail(t, "Test timed out")
			return
		}
		if assert.Nil(t, err1) && assert.Nil(t, err2) && notif1 && notif2 {
			return
		}
	}
}
