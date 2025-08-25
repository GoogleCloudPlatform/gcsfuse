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
	"crypto/rand"
	"os"
	"path"
	"testing"
	"time"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func createObjectInBucket(t *testing.T, objPath string, objSize int64, bucket gcs.Bucket) []byte {
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
	return objectContent
}

func configureFakeStorage(t *testing.T) storage.StorageHandle {
	t.Helper()
	mockClient := new(storage.MockStorageControlClient)
	fakeStorage := storage.NewFakeStorageWithMockClient(mockClient, cfg.HTTP2)
	t.Cleanup(func() { fakeStorage.ShutDown() })
	mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(&controlpb.StorageLayout{}, nil)
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

func createObjectInStoreAndInitCache(t *testing.T, cache *lru.Cache, bucket gcs.Bucket, objectName string, objectSize int64) (gcs.MinObject, []byte) {
	t.Helper()
	content := createObjectInBucket(t, objectName, objectSize, bucket)
	minObj := getMinObject(objectName, bucket)
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
	return minObj, content
}

func TestParallelDownloads(t *testing.T) {
	tbl := []struct {
		name                     string
		objectSize               int64
		readReqSize              int64
		parallelDownloadsPerFile int64
		maxParallelDownloads     int64
		downloadOffset           int64
		subscribedOffset         int64
		enableODirect            bool
	}{
		{
			name:                     "download the entire object when object size > no of goroutines * readReqSize",
			objectSize:               15 * util.MiB,
			readReqSize:              3,
			parallelDownloadsPerFile: 100,
			maxParallelDownloads:     3,
			subscribedOffset:         7,
			downloadOffset:           10,
			enableODirect:            true,
		},
		{
			name:                     "download only upto the object size",
			objectSize:               10 * util.MiB,
			readReqSize:              4,
			parallelDownloadsPerFile: 100,
			maxParallelDownloads:     3,
			subscribedOffset:         7,
			downloadOffset:           10,
			enableODirect:            true,
		},
		{
			name:                     "download the entire object with O_DIRECT disabled.",
			objectSize:               16 * util.MiB,
			readReqSize:              4,
			parallelDownloadsPerFile: 100,
			maxParallelDownloads:     3,
			subscribedOffset:         7,
			downloadOffset:           10,
			enableODirect:            false,
		},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc
			t.Parallel()
			cache, cacheDir := configureCache(t, 2*tc.objectSize)
			storageHandle := configureFakeStorage(t)
			ctx := context.Background()
			bucket, err := storageHandle.BucketHandle(ctx, storage.TestBucketName, "")
			assert.Nil(t, err)
			minObj, content := createObjectInStoreAndInitCache(t, cache, bucket, "path/in/gcs/foo.txt", tc.objectSize)
			fileCacheConfig := &cfg.FileCacheConfig{
				EnableParallelDownloads:  true,
				ParallelDownloadsPerFile: tc.parallelDownloadsPerFile,
				DownloadChunkSizeMb:      tc.readReqSize, EnableCrc: true,
				MaxParallelDownloads: tc.maxParallelDownloads,
				WriteBufferSize:      4 * 1024 * 1024,
				EnableODirect:        tc.enableODirect,
			}
			jm := NewJobManager(cache, util.DefaultFilePerm, util.DefaultDirPerm, cacheDir, 2, fileCacheConfig, metrics.NewNoopMetrics())
			job := jm.CreateJobIfNotExists(&minObj, bucket)
			subscriberC := job.subscribe(tc.subscribedOffset)

			_, err = job.Download(context.Background(), 10, false)

			timeout := time.After(1 * time.Second)
			for {
				select {
				case jobStatus := <-subscriberC:
					if assert.Nil(t, err) {
						require.GreaterOrEqual(t, tc.objectSize, jobStatus.Offset)
						verifyFileTillOffset(t,
							data.FileSpec{Path: util.GetDownloadPath(path.Join(cacheDir, storage.TestBucketName), "path/in/gcs/foo.txt"), FilePerm: util.DefaultFilePerm, DirPerm: util.DefaultDirPerm}, jobStatus.Offset,
							content)
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
	ctx := context.Background()
	bucket, err := storageHandle.BucketHandle(ctx, storage.TestBucketName, "")
	assert.Nil(t, err)
	minObj1, content1 := createObjectInStoreAndInitCache(t, cache, bucket, "path/in/gcs/foo.txt", 10*util.MiB)
	minObj2, content2 := createObjectInStoreAndInitCache(t, cache, bucket, "path/in/gcs/bar.txt", 5*util.MiB)
	fileCacheConfig := &cfg.FileCacheConfig{
		EnableParallelDownloads:  true,
		ParallelDownloadsPerFile: 100,
		DownloadChunkSizeMb:      2,
		EnableCrc:                true,
		MaxParallelDownloads:     2,
		WriteBufferSize:          4 * 1024 * 1024,
	}
	jm := NewJobManager(cache, util.DefaultFilePerm, util.DefaultDirPerm, cacheDir, 2, fileCacheConfig, metrics.NewNoopMetrics())
	job1 := jm.CreateJobIfNotExists(&minObj1, bucket)
	job2 := jm.CreateJobIfNotExists(&minObj2, bucket)
	s1 := job1.subscribe(10 * util.MiB)
	s2 := job2.subscribe(5 * util.MiB)

	_, err1 := job1.Download(ctx, 10*util.MiB, false)
	_, err2 := job2.Download(ctx, 5*util.MiB, false)

	notif1, notif2 := false, false
	timeout := time.After(1 * time.Second)
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
			verifyFileTillOffset(t,
				data.FileSpec{Path: util.GetDownloadPath(path.Join(cacheDir, storage.TestBucketName), "path/in/gcs/foo.txt"), FilePerm: util.DefaultFilePerm, DirPerm: util.DefaultDirPerm},
				10*util.MiB, content1)
			verifyFileTillOffset(t,
				data.FileSpec{Path: util.GetDownloadPath(path.Join(cacheDir, storage.TestBucketName), "path/in/gcs/bar.txt"), FilePerm: util.DefaultFilePerm, DirPerm: util.DefaultDirPerm},
				5*util.MiB, content2)
			return
		}
	}
}
