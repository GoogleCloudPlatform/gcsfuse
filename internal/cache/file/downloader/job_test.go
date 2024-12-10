// Copyright 2023 Google LLC
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
	"math"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/sync/semaphore"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const CacheMaxSize = 50
const DefaultObjectName = "foo"
const DefaultSequentialReadSizeMb = 100

func (dt *downloaderTest) initJobTest(objectName string, objectContent []byte, sequentialReadSize int32, lruCacheSize uint64, removeCallback func()) {
	ctx := context.Background()
	objects := map[string][]byte{objectName: objectContent}
	err := storageutil.CreateObjects(ctx, dt.bucket, objects)
	AssertEq(nil, err)
	dt.object = getMinObject(objectName, dt.bucket)
	dt.fileSpec = data.FileSpec{
		Path:     dt.fileCachePath(dt.bucket.Name(), dt.object.Name),
		FilePerm: util.DefaultFilePerm,
		DirPerm:  util.DefaultDirPerm,
	}
	dt.cache = lru.NewCache(lruCacheSize)

	dt.job = NewJob(&dt.object, dt.bucket, dt.cache, sequentialReadSize, dt.fileSpec, removeCallback, dt.defaultFileCacheConfig, semaphore.NewWeighted(math.MaxInt64), common.NewNoopMetrics())
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

func (dt *downloaderTest) verifyInvalidError(err error) {
	AssertTrue((nil == err) || (errors.Is(err, context.Canceled)) || (strings.Contains(err.Error(), lru.EntryNotExistErrMsg)),
		fmt.Sprintf("actual error:%v is not as expected", err))
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
	fileInfo := dt.getFileInfo()
	AssertTrue(fileInfo != nil)
	AssertEq(dt.object.Generation, fileInfo.(data.FileInfo).ObjectGeneration)
	AssertLe(offset, fileInfo.(data.FileInfo).Offset)
	AssertEq(dt.object.Size, fileInfo.(data.FileInfo).Size())
}

func (dt *downloaderTest) getFileInfo() lru.ValueType {
	fileInfoKey := data.FileInfoKey{BucketName: dt.bucket.Name(), ObjectName: dt.object.Name}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	return dt.cache.LookUp(fileInfoKeyName)
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

func (dt *downloaderTest) Test_updateStatusAndNotifySubscribers() {
	// Failed with error
	subscriberOffset1 := int64(3)
	subscriberOffset2 := int64(5)
	notificationC1 := dt.job.subscribe(subscriberOffset1)
	notificationC2 := dt.job.subscribe(subscriberOffset2)
	dt.job.status = JobStatus{Name: Downloading, Err: nil, Offset: 4}

	customErr := fmt.Errorf("custom error")
	dt.job.updateStatusAndNotifySubscribers(Failed, customErr)

	AssertEq(0, dt.job.subscribers.Len())
	notification1, ok1 := <-notificationC1
	notification2, ok2 := <-notificationC2
	jobStatus := JobStatus{Name: Failed, Err: customErr, Offset: 4}
	// Check 1st and 2nd subscriber notifications
	AssertTrue(reflect.DeepEqual(jobStatus, notification1))
	AssertEq(true, ok1)
	AssertTrue(reflect.DeepEqual(jobStatus, notification2))
	AssertEq(true, ok2)

	// Complete without error
	subscriberOffset1 = int64(3)
	notificationC1 = dt.job.subscribe(subscriberOffset1)
	dt.job.status = JobStatus{Name: Downloading, Err: nil, Offset: 4}

	dt.job.updateStatusAndNotifySubscribers(Completed, nil)

	AssertEq(0, dt.job.subscribers.Len())
	notification1, ok1 = <-notificationC1
	jobStatus = JobStatus{Name: Completed, Err: nil, Offset: 4}
	// Check subscriber notification
	AssertTrue(reflect.DeepEqual(jobStatus, notification1))
	AssertEq(true, ok1)
}

func (dt *downloaderTest) Test_updateStatusOffset_UpdateEntry() {
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
	dt.job.mu.Lock()
	dt.job.status.Name = Downloading
	notificationCh := dt.job.subscribe(5)
	dt.job.mu.Unlock()

	err = dt.job.updateStatusOffset(10)

	AssertEq(nil, err)
	// Confirm fileInfoCache is updated with new offset.
	lookupResult := dt.cache.LookUp(fileInfoKeyName)
	AssertFalse(lookupResult == nil)
	fileInfo = lookupResult.(data.FileInfo)
	AssertEq(10, fileInfo.Offset)
	AssertEq(dt.job.object.Generation, fileInfo.ObjectGeneration)
	AssertEq(dt.job.object.Size, fileInfo.FileSize)
	// Confirm job's status offset
	AssertEq(10, dt.job.status.Offset)
	// Check the subscriber's notification
	notification, ok := <-notificationCh
	AssertEq(true, ok)
	jobStatus := JobStatus{Name: Downloading, Err: nil, Offset: 10}
	AssertTrue(reflect.DeepEqual(jobStatus, notification))
}

func (dt *downloaderTest) Test_updateStatusOffset_InsertNew() {
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: dt.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	value := dt.cache.Erase(fileInfoKeyName)
	AssertTrue(value != nil)

	err = dt.job.updateStatusOffset(10)

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), lru.EntryNotExistErrMsg))
	// Confirm job's status offset
	AssertEq(0, dt.job.status.Offset)
}

func (dt *downloaderTest) Test_updateStatusOffset_Fail() {
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

	// Change the size of object and then try to update file info cache.
	dt.job.object.Size = 10
	err = dt.job.updateStatusOffset(15)

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), lru.InvalidUpdateEntrySizeErrorMsg))
	// Confirm job's status offset
	AssertEq(0, dt.job.status.Offset)
}

func (dt *downloaderTest) Test_cleanUpDownloadAsyncJob() {
	dt.job.mu.Lock()
	var callbackExecuted atomic.Bool
	removeCallback := func() { callbackExecuted.Store(true) }
	dt.job.removeJobCallback = removeCallback
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	dt.job.cancelCtx, dt.job.cancelFunc = cancelCtx, cancelFunc
	dt.job.mu.Unlock()

	dt.job.cleanUpDownloadAsyncJob()

	// Verify context is canceled
	AssertTrue(errors.Is(cancelCtx.Err(), context.Canceled))
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	// doneCh is closed
	_, ok := <-dt.job.doneCh
	AssertFalse(ok)
	// References to context and cancel function are removed
	AssertEq(nil, dt.job.cancelCtx)
	AssertEq(nil, dt.job.cancelFunc)
	AssertEq(nil, dt.job.removeJobCallback)
	// Call back function should have been called
	AssertTrue(callbackExecuted.Load())
}

func (dt *downloaderTest) Test_downloadObjectToFile() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 10 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})
	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())
	// Add subscriber
	subscribedOffset := int64(6 * util.MiB)
	notificationC := dt.job.subscribe(subscribedOffset)
	file, err := util.CreateFile(data.FileSpec{Path: dt.job.fileSpec.Path,
		FilePerm: os.FileMode(0600), DirPerm: os.FileMode(0700)}, os.O_TRUNC|os.O_RDWR)
	AssertEq(nil, err)
	defer func() {
		_ = file.Close()
	}()

	// Start download
	err = dt.job.downloadObjectToFile(file)

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

func (dt *downloaderTest) Test_downloadObjectToFile_CtxCancelled() {
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
	err = dt.job.downloadObjectToFile(file)

	AssertTrue(errors.Is(err, context.Canceled), fmt.Sprintf("didn't get context canceled error: %v", err))
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
	dt.waitForCrcCheckToBeCompleted()
	AssertEq(Completed, dt.job.status.Name)

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
	dt.verifyInvalidError(jobStatus.Err)
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
	objectName := "path/in/gcs/cancel.txt"
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
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2), nil)
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
	dt.waitForCrcCheckToBeCompleted()

	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	expectedJobStatus := JobStatus{Completed, nil, int64(objectSize)}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, dt.job.status))
	// Verify file
	dt.verifyFile(objectContent)
	// Verify file info cache
	dt.verifyFileInfoEntry(uint64(objectSize))
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

	AssertTrue((jobStatus.Name == Downloading) || (jobStatus.Name == Completed))
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, 0)
	// Verify file
	dt.verifyFile(objectContent[:jobStatus.Offset])
}

func (dt *downloaderTest) Test_Invalidate_WhenDownloading() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 10 * util.MiB
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
	dt.verifyInvalidError(dt.job.status.Err)
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
	dt.waitForCrcCheckToBeCompleted()
	jobStatus := dt.job.GetStatus()
	AssertEq(Completed, jobStatus.Name)

	dt.job.Invalidate()

	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertEq(Invalid, dt.job.status.Name)
	dt.verifyInvalidError(dt.job.status.Err)
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
		dt.verifyInvalidError(currJobStatus.Err)
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
		dt.verifyInvalidError(currJobStatus.Err)
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

func (dt *downloaderTest) Test_validateCRC_ForTamperedFileWhenEnableCRCIsTrue() {
	objectName := "path/in/gcs/file1.txt"
	objectSize := 8 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})
	// Start download
	offset := int64(8 * util.MiB)
	jobStatus, err := dt.job.Download(context.Background(), offset, true)
	AssertEq(nil, err)
	// Here the crc check will be successful
	dt.waitForCrcCheckToBeCompleted()
	AssertEq(Completed, dt.job.status.Name)
	AssertEq(nil, dt.job.status.Err)
	AssertGe(dt.job.status.Offset, offset)
	// Verify file
	dt.verifyFile(objectContent[:jobStatus.Offset])
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(jobStatus.Offset))
	// Tamper the file
	err = os.WriteFile(dt.fileSpec.Path, []byte("test"), 0644)
	AssertEq(nil, err)

	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())
	err = dt.job.validateCRC()

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "checksum mismatch detected"))
	AssertEq(nil, dt.getFileInfo())
	_, err = os.Stat(dt.fileSpec.Path)
	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

func (dt *downloaderTest) Test_validateCRC_ForTamperedFileWhenEnableCRCIsFalse() {
	objectName := "path/in/gcs/file2.txt"
	objectSize := 1 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})
	// Start download
	offset := int64(1 * util.MiB)
	jobStatus, err := dt.job.Download(context.Background(), offset, true)
	AssertEq(nil, err)
	// Here the crc check will be successful
	dt.waitForCrcCheckToBeCompleted()
	AssertEq(Completed, dt.job.status.Name)
	AssertEq(nil, dt.job.status.Err)
	AssertGe(dt.job.status.Offset, offset)
	// Verify file
	dt.verifyFile(objectContent[:jobStatus.Offset])
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(jobStatus.Offset))
	// Tamper the file
	err = os.WriteFile(dt.fileSpec.Path, []byte("test"), 0644)
	AssertEq(nil, err)
	dt.job.fileCacheConfig.EnableCrc = false

	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())
	err = dt.job.validateCRC()

	AssertEq(nil, err)
	// Verify file
	dt.verifyFile([]byte("test"))
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(jobStatus.Offset))
}

func (dt *downloaderTest) Test_validateCRC_WheContextIsCancelled() {
	objectName := "path/in/gcs/file2.txt"
	objectSize := 10 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})
	// Start download
	offset := int64(10 * util.MiB)
	_, err := dt.job.Download(context.Background(), offset, true)
	AssertEq(nil, err)
	AssertTrue((dt.job.status.Name == Downloading) || (dt.job.status.Name == Completed), fmt.Sprintf("got job status: %v", dt.job.status.Name))
	AssertEq(nil, dt.job.status.Err)
	AssertGe(dt.job.status.Offset, offset)

	dt.job.cancelFunc()
	dt.waitForCrcCheckToBeCompleted()

	AssertEq(Invalid, dt.job.status.Name)
	dt.verifyInvalidError(dt.job.status.Err)
}

func (dt *downloaderTest) Test_handleError_SetStatusAsInvalidWhenContextIsCancelled() {
	subscriberOffset := int64(1)
	notificationC := dt.job.subscribe(subscriberOffset)
	err := errors.Join(context.Canceled)

	err = fmt.Errorf("Wrapping with custom message %w", err)
	dt.job.handleError(err)

	AssertEq(0, dt.job.subscribers.Len())
	notification, ok := <-notificationC
	AssertEq(Invalid, notification.Name)
	dt.verifyInvalidError(notification.Err)
	AssertEq(0, notification.Offset)
	AssertEq(true, ok)
}

func (dt *downloaderTest) Test_handleError_SetStatusAsErrorWhenContextIsNotCancelled() {
	subscriberOffset := int64(1)
	notificationC := dt.job.subscribe(subscriberOffset)
	err := errors.New("custom error")

	updatedErr := fmt.Errorf("Custom message %w", err)
	dt.job.handleError(updatedErr)

	AssertEq(0, dt.job.subscribers.Len())
	notification, ok := <-notificationC
	jobStatus := JobStatus{Name: Failed, Err: updatedErr, Offset: 0}
	fmt.Println(notification)
	AssertTrue(reflect.DeepEqual(jobStatus, notification))
	AssertEq(true, ok)
}

func (dt *downloaderTest) Test_When_Parallel_Download_Is_Enabled() {
	//Arrange - initJobTest is being called in setup of downloader.go
	dt.job.fileCacheConfig.EnableParallelDownloads = true

	result := dt.job.IsParallelDownloadsEnabled()

	AssertTrue(result)
}

func (dt *downloaderTest) Test_When_Parallel_Download_Is_Disabled() {
	//Arrange - initJobTest is being called in setup of downloader.go
	dt.job.fileCacheConfig.EnableParallelDownloads = false

	result := dt.job.IsParallelDownloadsEnabled()

	AssertFalse(result)
}

func (dt *downloaderTest) Test_createCacheFile_WhenNonParallelDownloads() {
	//Arrange - initJobTest is being called in setup of downloader.go
	dt.job.fileCacheConfig.EnableParallelDownloads = false

	cacheFile, err := dt.job.createCacheFile()

	AssertEq(nil, err)
	defer func() {
		_ = cacheFile.Close()
	}()
}

func (dt *downloaderTest) Test_createCacheFile_WhenParallelDownloads() {
	//Arrange - initJobTest is being called in setup of downloader.go
	dt.job.fileCacheConfig.EnableParallelDownloads = true

	cacheFile, err := dt.job.createCacheFile()

	AssertEq(nil, err)
	defer func() {
		_ = cacheFile.Close()
	}()
}

func (dt *downloaderTest) Test_createCacheFile_WhenParallelDownloadsEnabledAndODirectDisabled() {
	//Arrange - initJobTest is being called in setup of downloader.go
	dt.job.fileCacheConfig.EnableParallelDownloads = true
	dt.job.fileCacheConfig.EnableODirect = false

	cacheFile, err := dt.job.createCacheFile()

	AssertEq(nil, err)
	defer func() {
		_ = cacheFile.Close()
	}()
}
