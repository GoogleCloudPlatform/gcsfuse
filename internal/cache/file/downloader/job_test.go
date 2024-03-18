// Copyright 2023 Google Inc. All Rights Reserved.
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
	"container/list"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const CacheMaxSize = 50
const DefaultObjectName = "foo"
const DefaultSequentialReadSizeMb = 100

func (dt *downloaderTest) getMinObject(objectName string) gcs.MinObject {
	ctx := context.Background()
	minObject, _, err := dt.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: objectName,
		ForceFetchFromGcs: true})
	if err != nil {
		panic(fmt.Errorf("error whlie stating object: %w", err))
	}

	if minObject != nil {
		return *minObject
	}
	return gcs.MinObject{}
}

func (dt *downloaderTest) initJobTest(objectName string, objectContent []byte, sequentialReadSize int32, lruCacheSize uint64, removeCallback func()) {
	ctx := context.Background()
	objects := map[string][]byte{objectName: objectContent}
	err := storageutil.CreateObjects(ctx, dt.bucket, objects)
	AssertEq(nil, err)
	dt.object = dt.getMinObject(objectName)
	dt.fileSpec = data.FileSpec{
		Path:     dt.fileCachePath(dt.bucket.Name(), dt.object.Name),
		FilePerm: util.DefaultFilePerm,
		DirPerm:  util.DefaultDirPerm,
	}
	dt.cache = lru.NewCache(lruCacheSize)
	dt.job = NewJob(&dt.object, dt.bucket, dt.cache, sequentialReadSize, dt.fileSpec, removeCallback)
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: objectName,
	}
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: dt.object.Generation,
		FileSize:         dt.object.Size,
		Offset:           0,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	_, err = dt.cache.Insert(fileInfoKeyName, fileInfo)
	AssertEq(nil, err)
}

func (dt *downloaderTest) verifyFile(content []byte) {
	fileStat, err := os.Stat(dt.fileSpec.Path)
	AssertEq(nil, err)
	AssertEq(dt.fileSpec.FilePerm, fileStat.Mode())
	AssertLe(len(content), fileStat.Size())
	// Verify the content of file downloaded only till the size of content passed.
	fileContent, err := os.ReadFile(dt.fileSpec.Path)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(content, fileContent[:len(content)]))
}

func (dt *downloaderTest) verifyFileInfoEntry(offset uint64) {
	fileInfoKey := data.FileInfoKey{BucketName: dt.bucket.Name(), ObjectName: dt.object.Name}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	fileInfo := dt.cache.LookUp(fileInfoKeyName)
	AssertTrue(fileInfo != nil)
	AssertEq(dt.object.Generation, fileInfo.(data.FileInfo).ObjectGeneration)
	AssertLe(offset, fileInfo.(data.FileInfo).Offset)
	AssertEq(dt.object.Size, fileInfo.(data.FileInfo).Size())
}

func (dt *downloaderTest) fileCachePath(bucketName string, objectName string) string {
	return path.Join(cacheDir, bucketName, objectName)
}

func (dt *downloaderTest) Test_init() {
	// Explicitly changing initialized values first.
	dt.job.status.Name = Downloading
	dt.job.status.Err = fmt.Errorf("some error")
	dt.job.status.Offset = -1
	dt.job.subscribers.PushBack(struct{}{})

	dt.job.init()

	AssertEq(NotStarted, dt.job.status.Name)
	AssertEq(nil, dt.job.status.Err)
	AssertEq(0, dt.job.status.Offset)
	AssertTrue(reflect.DeepEqual(list.List{}, dt.job.subscribers))
}

func (dt *downloaderTest) Test_subscribe() {
	subscriberOffset1 := int64(0)
	subscriberOffset2 := int64(1)

	notificationC1 := dt.job.subscribe(subscriberOffset1)
	notificationC2 := dt.job.subscribe(subscriberOffset2)

	AssertEq(2, dt.job.subscribers.Len())
	receivingC := make(<-chan JobStatus, 1)
	AssertEq(reflect.TypeOf(receivingC), reflect.TypeOf(notificationC1))
	AssertEq(reflect.TypeOf(receivingC), reflect.TypeOf(notificationC2))
	// Check 1st and 2nd subscribers
	var subscriber jobSubscriber
	AssertEq(reflect.TypeOf(subscriber), reflect.TypeOf(dt.job.subscribers.Front().Value.(jobSubscriber)))
	AssertEq(0, dt.job.subscribers.Front().Value.(jobSubscriber).subscribedOffset)
	AssertEq(reflect.TypeOf(subscriber), reflect.TypeOf(dt.job.subscribers.Back().Value.(jobSubscriber)))
	AssertEq(1, dt.job.subscribers.Back().Value.(jobSubscriber).subscribedOffset)
}

func (dt *downloaderTest) Test_notifySubscriber_Failed() {
	subscriberOffset := int64(1)
	notificationC := dt.job.subscribe(subscriberOffset)
	dt.job.status.Name = Failed
	customErr := fmt.Errorf("custom err")
	dt.job.status.Err = customErr

	dt.job.notifySubscribers()

	AssertEq(0, dt.job.subscribers.Len())
	notification, ok := <-notificationC
	jobStatus := JobStatus{Name: Failed, Err: customErr, Offset: 0}
	AssertTrue(reflect.DeepEqual(jobStatus, notification))
	AssertEq(true, ok)
}

func (dt *downloaderTest) Test_notifySubscriber_Invalid() {
	subscriberOffset := int64(1)
	notificationC := dt.job.subscribe(subscriberOffset)
	dt.job.status.Name = Invalid

	dt.job.notifySubscribers()

	AssertEq(0, dt.job.subscribers.Len())
	notification, ok := <-notificationC
	jobStatus := JobStatus{Name: Invalid, Err: nil, Offset: 0}
	AssertTrue(reflect.DeepEqual(jobStatus, notification))
	AssertEq(true, ok)
}

func (dt *downloaderTest) Test_notifySubscriber_SubscribedOffset() {
	subscriberOffset1 := int64(3)
	subscriberOffset2 := int64(5)
	notificationC1 := dt.job.subscribe(subscriberOffset1)
	_ = dt.job.subscribe(subscriberOffset2)
	dt.job.status.Name = Downloading
	dt.job.status.Offset = 4

	dt.job.notifySubscribers()

	AssertEq(1, dt.job.subscribers.Len())
	notification1, ok := <-notificationC1
	jobStatus := JobStatus{Name: Downloading, Err: nil, Offset: 4}
	AssertTrue(reflect.DeepEqual(jobStatus, notification1))
	AssertEq(true, ok)
	// Check 2nd subscriber's offset
	AssertEq(subscriberOffset2, dt.job.subscribers.Front().Value.(jobSubscriber).subscribedOffset)
}

func (dt *downloaderTest) Test_failWhileDownloading() {
	subscriberOffset1 := int64(3)
	subscriberOffset2 := int64(5)
	notificationC1 := dt.job.subscribe(subscriberOffset1)
	notificationC2 := dt.job.subscribe(subscriberOffset2)
	dt.job.status = JobStatus{Name: Downloading, Err: nil, Offset: 4}

	customErr := fmt.Errorf("custom error")
	dt.job.failWhileDownloading(customErr)

	AssertEq(0, dt.job.subscribers.Len())
	notification1, ok1 := <-notificationC1
	notification2, ok2 := <-notificationC2
	jobStatus := JobStatus{Name: Failed, Err: customErr, Offset: 4}
	// Check 1st and 2nd subscriber notifications
	AssertTrue(reflect.DeepEqual(jobStatus, notification1))
	AssertEq(true, ok1)
	AssertTrue(reflect.DeepEqual(jobStatus, notification2))
	AssertEq(true, ok2)
}

func (dt *downloaderTest) Test_updateFileInfoCache_UpdateEntry() {
	// Add an entry into
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: DefaultObjectName,
	}
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: dt.job.object.Generation,
		FileSize:         dt.job.object.Size,
		Offset:           0,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	_, err = dt.cache.Insert(fileInfoKeyName, fileInfo)
	AssertEq(nil, err)
	dt.job.status.Offset = 1

	err = dt.job.updateFileInfoCache()

	AssertEq(nil, err)
	// Confirm fileInfoCache is updated with new offset.
	lookupResult := dt.cache.LookUp(fileInfoKeyName)
	AssertFalse(lookupResult == nil)
	fileInfo = lookupResult.(data.FileInfo)
	AssertEq(1, fileInfo.Offset)
	AssertEq(dt.job.object.Generation, fileInfo.ObjectGeneration)
	AssertEq(dt.job.object.Size, fileInfo.FileSize)
}

func (dt *downloaderTest) Test_updateFileInfoCache_InsertNew() {
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: dt.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	value := dt.cache.Erase(fileInfoKeyName)
	AssertTrue(value != nil)

	err = dt.job.updateFileInfoCache()

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), lru.EntryNotExistErrMsg))
}

func (dt *downloaderTest) Test_updateFileInfoCache_Fail() {
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: DefaultObjectName,
	}
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: dt.job.object.Generation,
		FileSize:         dt.job.object.Size,
		Offset:           0,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	_, err = dt.cache.Insert(fileInfoKeyName, fileInfo)
	AssertEq(nil, err)

	// change the size of object and then try to update file info cache.
	dt.job.object.Size = 10
	err = dt.job.updateFileInfoCache()

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), lru.InvalidUpdateEntrySizeErrorMsg))
}

// Note: We can't test Test_downloadObjectAsync_MoreThanSequentialReadSize as
// the fake storage bucket/server in the testing environment doesn't support
// reading ranges (start and limit in NewReader call)
func (dt *downloaderTest) Test_downloadObjectAsync_LessThanSequentialReadSize() {
	// Create new object in bucket and create new job for it.
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecuted atomic.Bool
	removeCallback := func() { callbackExecuted.Store(true) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), removeCallback)
	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())

	// Start download
	dt.job.downloadObjectAsync()

	// Check job completed successfully
	jobStatus := JobStatus{Completed, nil, int64(objectSize)}
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertTrue(reflect.DeepEqual(jobStatus, dt.job.status))
	// Verify file is downloaded
	dt.verifyFile(objectContent)
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(objectSize))
	// Verify callback is executed and removed
	AssertTrue(callbackExecuted.Load())
	AssertEq(nil, dt.job.removeJobCallback)
}

func (dt *downloaderTest) Test_downloadObjectAsync_LessThanChunkSize() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 2 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecuted atomic.Bool
	removeCallback := func() { callbackExecuted.Store(true) }
	dt.initJobTest(objectName, objectContent, 25, uint64(2*objectSize), removeCallback)
	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())

	// start download
	dt.job.downloadObjectAsync()

	// check job completed successfully
	jobStatus := JobStatus{Completed, nil, int64(objectSize)}
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertTrue(reflect.DeepEqual(jobStatus, dt.job.status))
	// Verify file is downloaded
	dt.verifyFile(objectContent)
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(objectSize))
	// Verify callback is executed and removed
	AssertTrue(callbackExecuted.Load())
	AssertEq(nil, dt.job.removeJobCallback)
}

func (dt *downloaderTest) Test_downloadObjectAsync_Notification() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecuted atomic.Bool
	removeCallback := func() { callbackExecuted.Store(true) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), removeCallback)
	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())
	// Add subscriber
	subscribedOffset := int64(9 * util.MiB)
	notificationC := dt.job.subscribe(subscribedOffset)

	// start download
	dt.job.downloadObjectAsync()

	jobStatus := <-notificationC
	// check the notification is sent after subscribed offset
	AssertGe(jobStatus.Offset, subscribedOffset)
	// check job completed successfully
	jobStatus = JobStatus{Completed, nil, int64(objectSize)}
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertTrue(reflect.DeepEqual(jobStatus, dt.job.status))
	// verify file is downloaded
	dt.verifyFile(objectContent)
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(objectSize))
	// Verify callback is executed and removed
	AssertTrue(callbackExecuted.Load())
	AssertEq(nil, dt.job.removeJobCallback)
}

func (dt *downloaderTest) Test_Download_WhenNotStarted() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 16 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})

	// Start download
	offset := int64(8 * util.MiB)
	jobStatus, err := dt.job.Download(context.Background(), offset, true)

	AssertEq(nil, err)
	// Verify that jobStatus is downloading and downloaded more than 8 Mib.
	AssertEq(Downloading, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, offset)
	// Verify file
	dt.verifyFile(objectContent[:jobStatus.Offset])
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(jobStatus.Offset))
}

func (dt *downloaderTest) Test_Download_WhenAlreadyDownloading() {
	// Create new object in bucket and create new job for it.
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})
	// Start download but not wait for download
	ctx := context.Background()
	jobStatus, err := dt.job.Download(ctx, 1, false)
	AssertEq(nil, err)
	AssertEq(Downloading, jobStatus.Name)

	// Again call download but wait for download this time.
	offset := int64(25 * util.MiB)
	jobStatus, err = dt.job.Download(ctx, offset, true)

	AssertEq(nil, err)
	// Verify that jobStatus is downloading and downloaded at least 25 Mib.
	AssertEq(Downloading, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, offset)
	// Verify file
	dt.verifyFile(objectContent[:jobStatus.Offset])
	// verify file info cache
	dt.verifyFileInfoEntry(uint64(jobStatus.Offset))
}

func (dt *downloaderTest) Test_Download_WhenAlreadyCompleted() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 16 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})
	// Wait for whole download to be completed.
	ctx := context.Background()
	jobStatus, err := dt.job.Download(ctx, int64(objectSize), true)
	AssertEq(nil, err)
	// Verify that jobStatus is Downloading but offset is object size
	expectedJobStatus := JobStatus{Downloading, nil, int64(objectSize)}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))

	// Try to request for some offset when job was already completed.
	offset := int64(16 * util.MiB)
	jobStatus, err = dt.job.Download(ctx, offset, true)

	AssertEq(nil, err)
	// Verify that jobStatus is completed & offset returned is still 16 MiB
	// this ensures that async job is not started again.
	AssertEq(Completed, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, objectSize)
	// Verify file
	dt.verifyFile(objectContent)
	// Verify file info cache
	dt.verifyFileInfoEntry(uint64(jobStatus.Offset))
}

func (dt *downloaderTest) Test_Download_WhenAsyncFails() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecuted atomic.Bool
	removeCallback := func() { callbackExecuted.Store(true) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize), removeCallback)
	// In real world, object size will not change in between, we are changing it
	// just to simulate the failure
	dt.job.object.Size = uint64(objectSize - 10)

	// Wait for whole download to be completed/failed.
	ctx := context.Background()
	jobStatus, err := dt.job.Download(ctx, int64(objectSize-10), true)

	AssertEq(nil, err)
	// Verify that jobStatus is failed
	AssertEq(Failed, jobStatus.Name)
	AssertGe(jobStatus.Offset, 0)
	AssertTrue(strings.Contains(jobStatus.Err.Error(), lru.InvalidUpdateEntrySizeErrorMsg))
	// Verify callback is executed
	AssertTrue(callbackExecuted.Load())
}

func (dt *downloaderTest) Test_Download_AlreadyFailed() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize), func() {})
	dt.job.mu.Lock()
	dt.job.status = JobStatus{Failed, fmt.Errorf(lru.InvalidUpdateEntrySizeErrorMsg), 8}
	dt.job.mu.Unlock()

	// Requesting again from download job which is in failed state
	jobStatus, err := dt.job.Download(context.Background(), int64(objectSize-1), true)

	AssertEq(nil, err)
	AssertEq(Failed, jobStatus.Name)
	AssertTrue(strings.Contains(jobStatus.Err.Error(), lru.InvalidUpdateEntrySizeErrorMsg))
}

func (dt *downloaderTest) Test_Download_AlreadyInvalid() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize), func() {})
	// Make the state as invalid
	dt.job.mu.Lock()
	dt.job.status.Name = Invalid
	dt.job.mu.Unlock()

	// Requesting download when already invalid.
	jobStatus, err := dt.job.Download(context.Background(), int64(objectSize), true)

	AssertEq(nil, err)
	AssertEq(Invalid, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
}

func (dt *downloaderTest) Test_Download_FileInfoRemovedInBetween() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 16 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecuted atomic.Bool
	removeCallback := func() { callbackExecuted.Store(true) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize), removeCallback)
	fileInfoKey := data.FileInfoKey{BucketName: dt.bucket.Name(), ObjectName: objectName}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		jobStatus, err := dt.job.Download(context.Background(), int64(objectSize), true)
		AssertEq(nil, err)
		AssertEq(Invalid, jobStatus.Name)
		wg.Done()
	}()

	// Delete fileinfo from file info cache
	dt.job.fileInfoCache.Erase(fileInfoKeyName)

	wg.Wait()
	AssertTrue(callbackExecuted.Load())
}

func (dt *downloaderTest) Test_Download_InvalidOffset() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecuted atomic.Bool
	removeCallback := func() { callbackExecuted.Store(true) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize), removeCallback)

	// requesting invalid offset
	offset := int64(objectSize) + 1
	jobStatus, err := dt.job.Download(context.Background(), offset, true)

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), fmt.Sprintf("Download: the requested offset %d is greater than the size of object %d", offset, dt.object.Size)))
	expectedJobStatus := JobStatus{NotStarted, nil, 0}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))
	AssertFalse(callbackExecuted.Load())
}

func (dt *downloaderTest) Test_Download_CtxCancelled() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 16 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecuted atomic.Bool
	removeCallback := func() { callbackExecuted.Store(true) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2), removeCallback)

	// Requesting full download and then the download call should be cancelled after
	// timeout but async download shouldn't be cancelled
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Millisecond*2)
	defer cancelFunc()
	offset := int64(objectSize)
	jobStatus, err := dt.job.Download(ctx, offset, true)

	AssertNe(nil, err)
	AssertTrue(errors.Is(err, context.DeadlineExceeded))
	// jobStatus is empty in this case.
	AssertEq("", jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	// job should be either Downloading or Completed.
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertTrue(dt.job.status.Name == Downloading || dt.job.status.Name == Completed)
	if dt.job.status.Name == Downloading {
		AssertFalse(callbackExecuted.Load())
	} else {
		AssertTrue(callbackExecuted.Load())
	}
	AssertEq(nil, dt.job.status.Err)
	// Verify file
	dt.verifyFile(objectContent[:dt.job.status.Offset])
	// Verify file info cache
	dt.verifyFileInfoEntry(uint64(dt.job.status.Offset))
}

func (dt *downloaderTest) Test_Download_Concurrent() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecutionCount atomic.Int32
	removeCallback := func() { callbackExecutionCount.Add(1) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2), removeCallback)
	ctx := context.Background()
	wg := sync.WaitGroup{}
	offsets := []int64{0, 4 * util.MiB, 16 * util.MiB, 8 * util.MiB, int64(objectSize), int64(objectSize) + 1}
	expectedErrs := []error{nil, nil, nil, nil, nil, fmt.Errorf(fmt.Sprintf("Download: the requested offset %d is greater than the size of object %d", int64(objectSize)+1, int64(objectSize)))}
	downloadFunc := func(expectedOffset int64, expectedErr error) {
		defer wg.Done()
		var jobStatus JobStatus
		var err error
		jobStatus, err = dt.job.Download(ctx, expectedOffset, true)
		AssertNe(Failed, jobStatus.Name)
		if expectedErr != nil {
			AssertTrue(strings.Contains(err.Error(), expectedErr.Error()))
			return
		} else {
			AssertEq(expectedErr, err)
		}
		AssertGe(jobStatus.Offset, expectedOffset)
	}

	// Start concurrent downloads
	for i, offset := range offsets {
		wg.Add(1)
		go downloadFunc(offset, expectedErrs[i])
	}
	wg.Wait()

	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	expectedJobStatus := JobStatus{Completed, nil, int64(objectSize)}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, dt.job.status))
	// Verify file
	dt.verifyFile(objectContent)
	// Verify file info cache
	dt.verifyFileInfoEntry(uint64(objectSize))
	// Verify callback is executed only once and removed
	AssertEq(1, callbackExecutionCount.Load())
	AssertEq(nil, dt.job.removeJobCallback)
}

func (dt *downloaderTest) Test_GetStatus() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 20 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2), func() {})
	ctx := context.Background()
	// Start download
	jobStatus, err := dt.job.Download(ctx, util.MiB, true)
	AssertEq(nil, err)
	AssertEq(Downloading, jobStatus.Name)

	// GetStatus in between downloading
	jobStatus = dt.job.GetStatus()

	AssertEq(Downloading, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, 0)
	// Verify file
	dt.verifyFile(objectContent[:jobStatus.Offset])
}

func (dt *downloaderTest) Test_Invalidate_WhenDownloading() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 8 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecuted atomic.Bool
	removeCallback := func() { callbackExecuted.Store(true) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2), removeCallback)
	ctx := context.Background()
	// Start download without waiting
	jobStatus, err := dt.job.Download(ctx, 0, false)
	AssertEq(nil, err)
	AssertEq(Downloading, jobStatus.Name)

	dt.job.Invalidate()

	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertEq(nil, dt.job.status.Err)
	AssertEq(Invalid, dt.job.status.Name)
	AssertTrue(callbackExecuted.Load())
	AssertEq(nil, dt.job.removeJobCallback)
}

func (dt *downloaderTest) Test_Invalidate_NotStarted() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecuted atomic.Bool
	removeCallback := func() { callbackExecuted.Store(true) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2), removeCallback)

	dt.job.Invalidate()

	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertEq(nil, dt.job.status.Err)
	AssertEq(Invalid, dt.job.status.Name)
	AssertTrue(callbackExecuted.Load())
	AssertEq(nil, dt.job.removeJobCallback)
}

func (dt *downloaderTest) Test_Invalidate_WhenAlreadyCompleted() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecutionCount atomic.Int32
	removeCallback := func() { callbackExecutionCount.Add(1) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2), removeCallback)
	ctx := context.Background()
	// Start download with waiting
	_, err := dt.job.Download(ctx, int64(objectSize), true)
	AssertEq(nil, err)
	jobStatus := dt.job.GetStatus()
	AssertEq(Completed, jobStatus.Name)

	dt.job.Invalidate()

	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertEq(nil, dt.job.status.Err)
	AssertEq(Invalid, dt.job.status.Name)
	AssertEq(1, callbackExecutionCount.Load())
	AssertEq(nil, dt.job.removeJobCallback)
}

func (dt *downloaderTest) Test_Invalidate_Concurrent() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 20 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecutionCount atomic.Int32
	removeCallback := func() { callbackExecutionCount.Add(1) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2), removeCallback)
	ctx := context.Background()
	// Start download without waiting
	jobStatus, err := dt.job.Download(ctx, 0, false)
	AssertEq(nil, err)
	AssertEq(Downloading, jobStatus.Name)
	wg := sync.WaitGroup{}
	invalidateFunc := func() {
		defer wg.Done()
		dt.job.Invalidate()
		currJobStatus := dt.job.GetStatus()
		AssertEq(Invalid, currJobStatus.Name)
		AssertEq(nil, currJobStatus.Err)
	}

	// start concurrent Invalidate
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go invalidateFunc()
	}
	wg.Wait()

	AssertEq(1, callbackExecutionCount.Load())
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertEq(nil, dt.job.removeJobCallback)
}

func (dt *downloaderTest) Test_Invalidate_Download_Concurrent() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 10 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	var callbackExecutionCount atomic.Int32
	removeCallback := func() { callbackExecutionCount.Add(1) }
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2), removeCallback)
	wg := sync.WaitGroup{}
	downloadFunc := func(offset int64, waitForDownload bool) {
		defer wg.Done()
		ctx := context.Background()
		// Start download without waiting
		jobStatus, err := dt.job.Download(ctx, offset, waitForDownload)
		AssertEq(nil, err)
		AssertTrue(jobStatus.Name == Downloading || jobStatus.Name == Invalid || jobStatus.Name == Completed)
		// If status is downloading/complete and wait for download is true then
		// status offset should be at least requested offset.
		if waitForDownload && (jobStatus.Name == Downloading || jobStatus.Name == Completed) {
			AssertGe(jobStatus.Offset, offset)
		}
	}
	invalidateFunc := func() {
		defer wg.Done()
		dt.job.Invalidate()
		currJobStatus := dt.job.GetStatus()
		AssertEq(Invalid, currJobStatus.Name)
		AssertEq(nil, currJobStatus.Err)
	}

	// Start concurrent invalidate and download
	offsets := [6]int64{0, util.MiB, 5 * util.MiB, 0, 2 * util.MiB, 10 * util.MiB}
	for i := 0; i < len(offsets); i++ {
		wg.Add(2)
		waitForDownload := false
		if i%2 == 0 {
			waitForDownload = true
		}
		go downloadFunc(offsets[i], waitForDownload)
		go invalidateFunc()
	}
	wg.Wait()

	AssertEq(1, callbackExecutionCount.Load())
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertEq(nil, dt.job.removeJobCallback)
}
