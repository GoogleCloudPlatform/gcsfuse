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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/storageutil"
	testutil "github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"github.com/stretchr/testify/assert"
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
	assert.Nil(dt.T(), err)
	dt.object = getMinObject(objectName, dt.bucket)
	dt.fileSpec = data.FileSpec{
		Path:     dt.fileCachePath(dt.bucket.Name(), dt.object.Name),
		FilePerm: util.DefaultFilePerm,
		DirPerm:  util.DefaultDirPerm,
	}
	dt.cache = lru.NewCache(lruCacheSize)

	dt.job = NewJob(&dt.object, dt.bucket, dt.cache, sequentialReadSize, dt.fileSpec, removeCallback, dt.defaultFileCacheConfig, semaphore.NewWeighted(math.MaxInt64))
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
	assert.Nil(dt.T(), err)
	_, err = dt.cache.Insert(fileInfoKeyName, fileInfo)
	assert.Nil(dt.T(), err)
}

func (dt *downloaderTest) verifyInvalidError(err error) {
	assert.True(dt.T(), (nil == err) || (errors.Is(err, context.Canceled)) || (strings.Contains(err.Error(), lru.EntryNotExistErrMsg)),
		fmt.Sprintf("actual error:%v is not as expected", err))
}

func (dt *downloaderTest) verifyFile(content []byte) {
	fileStat, err := os.Stat(dt.fileSpec.Path)
	assert.Nil(dt.T(), err)
	assert.Equal(dt.T(), dt.fileSpec.FilePerm, fileStat.Mode())
	assert.LessOrEqual(dt.T(), int64(len(content)), fileStat.Size())
	// Verify the content of file downloaded only till the size of content passed.
	fileContent, err := os.ReadFile(dt.fileSpec.Path)
	assert.Nil(dt.T(), err)
	assert.True(dt.T(), reflect.DeepEqual(content, fileContent[:len(content)]))
}

func (dt *downloaderTest) verifyFileInfoEntry(offset uint64) {
	fileInfo := dt.getFileInfo()
	assert.True(dt.T(), fileInfo != nil)
	assert.Equal(dt.T(), dt.object.Generation, fileInfo.(data.FileInfo).ObjectGeneration)
	assert.LessOrEqual(dt.T(), offset, fileInfo.(data.FileInfo).Offset)
	assert.Equal(dt.T(), dt.object.Size, fileInfo.(data.FileInfo).Size())
}

func (dt *downloaderTest) getFileInfo() lru.ValueType {
	fileInfoKey := data.FileInfoKey{BucketName: dt.bucket.Name(), ObjectName: dt.object.Name}
	fileInfoKeyName, err := fileInfoKey.Key()
	assert.Nil(dt.T(), err)
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

	assert.Equal(dt.T(), NotStarted, dt.job.status.Name)
	assert.Nil(dt.T(), dt.job.status.Err)
	assert.EqualValues(dt.T(), 0, dt.job.status.Offset)
	assert.True(dt.T(), reflect.DeepEqual(list.List{}, dt.job.subscribers))
}

func (dt *downloaderTest) Test_subscribe() {
	subscriberOffset1 := int64(0)
	subscriberOffset2 := int64(1)

	notificationC1 := dt.job.subscribe(subscriberOffset1)
	notificationC2 := dt.job.subscribe(subscriberOffset2)

	assert.Equal(dt.T(), 2, dt.job.subscribers.Len())
	receivingC := make(<-chan JobStatus, 1)
	assert.Equal(dt.T(), reflect.TypeOf(receivingC), reflect.TypeOf(notificationC1))
	assert.Equal(dt.T(), reflect.TypeOf(receivingC), reflect.TypeOf(notificationC2))
	// Check 1st and 2nd subscribers
	var subscriber jobSubscriber
	assert.Equal(dt.T(), reflect.TypeOf(subscriber), reflect.TypeOf(dt.job.subscribers.Front().Value.(jobSubscriber)))
	assert.EqualValues(dt.T(), 0, dt.job.subscribers.Front().Value.(jobSubscriber).subscribedOffset)
	assert.Equal(dt.T(), reflect.TypeOf(subscriber), reflect.TypeOf(dt.job.subscribers.Back().Value.(jobSubscriber)))
	assert.EqualValues(dt.T(), 1, dt.job.subscribers.Back().Value.(jobSubscriber).subscribedOffset)
}

func (dt *downloaderTest) Test_notifySubscriber_Failed() {
	subscriberOffset := int64(1)
	notificationC := dt.job.subscribe(subscriberOffset)
	dt.job.status.Name = Failed
	customErr := fmt.Errorf("custom err")
	dt.job.status.Err = customErr

	dt.job.notifySubscribers()

	assert.Equal(dt.T(), 0, dt.job.subscribers.Len())
	notification, ok := <-notificationC
	jobStatus := JobStatus{Name: Failed, Err: customErr, Offset: 0}
	assert.True(dt.T(), reflect.DeepEqual(jobStatus, notification))
	assert.Equal(dt.T(), true, ok)
}

func (dt *downloaderTest) Test_notifySubscriber_Invalid() {
	subscriberOffset := int64(1)
	notificationC := dt.job.subscribe(subscriberOffset)
	dt.job.status.Name = Invalid

	dt.job.notifySubscribers()

	assert.Equal(dt.T(), 0, dt.job.subscribers.Len())
	notification, ok := <-notificationC
	jobStatus := JobStatus{Name: Invalid, Err: nil, Offset: 0}
	assert.True(dt.T(), reflect.DeepEqual(jobStatus, notification))
	assert.Equal(dt.T(), true, ok)
}

func (dt *downloaderTest) Test_notifySubscriber_SubscribedOffset() {
	subscriberOffset1 := int64(3)
	subscriberOffset2 := int64(5)
	notificationC1 := dt.job.subscribe(subscriberOffset1)
	_ = dt.job.subscribe(subscriberOffset2)
	dt.job.status.Name = Downloading
	dt.job.status.Offset = 4

	dt.job.notifySubscribers()

	assert.Equal(dt.T(), 1, dt.job.subscribers.Len())
	notification1, ok := <-notificationC1
	jobStatus := JobStatus{Name: Downloading, Err: nil, Offset: 4}
	assert.True(dt.T(), reflect.DeepEqual(jobStatus, notification1))
	assert.Equal(dt.T(), true, ok)
	// Check 2nd subscriber's offset
	assert.Equal(dt.T(), subscriberOffset2, dt.job.subscribers.Front().Value.(jobSubscriber).subscribedOffset)
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

	assert.Equal(dt.T(), 0, dt.job.subscribers.Len())
	notification1, ok1 := <-notificationC1
	notification2, ok2 := <-notificationC2
	jobStatus := JobStatus{Name: Failed, Err: customErr, Offset: 4}
	// Check 1st and 2nd subscriber notifications
	assert.True(dt.T(), reflect.DeepEqual(jobStatus, notification1))
	assert.Equal(dt.T(), true, ok1)
	assert.True(dt.T(), reflect.DeepEqual(jobStatus, notification2))
	assert.Equal(dt.T(), true, ok2)

	// Complete without error
	subscriberOffset1 = int64(3)
	notificationC1 = dt.job.subscribe(subscriberOffset1)
	dt.job.status = JobStatus{Name: Downloading, Err: nil, Offset: 4}

	dt.job.updateStatusAndNotifySubscribers(Completed, nil)

	assert.Equal(dt.T(), 0, dt.job.subscribers.Len())
	notification1, ok1 = <-notificationC1
	jobStatus = JobStatus{Name: Completed, Err: nil, Offset: 4}
	// Check subscriber notification
	assert.True(dt.T(), reflect.DeepEqual(jobStatus, notification1))
	assert.Equal(dt.T(), true, ok1)
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
	assert.Nil(dt.T(), err)
	_, err = dt.cache.Insert(fileInfoKeyName, fileInfo)
	assert.Nil(dt.T(), err)
	dt.job.mu.Lock()
	dt.job.status.Name = Downloading
	notificationCh := dt.job.subscribe(5)
	dt.job.mu.Unlock()

	err = dt.job.updateStatusOffset(10)

	assert.Nil(dt.T(), err)
	// Confirm fileInfoCache is updated with new offset.
	lookupResult := dt.cache.LookUp(fileInfoKeyName)
	assert.False(dt.T(), lookupResult == nil)
	fileInfo = lookupResult.(data.FileInfo)
	assert.EqualValues(dt.T(), 10, fileInfo.Offset)
	assert.Equal(dt.T(), dt.job.object.Generation, fileInfo.ObjectGeneration)
	assert.Equal(dt.T(), dt.job.object.Size, fileInfo.FileSize)
	// Confirm job's status offset
	assert.EqualValues(dt.T(), 10, dt.job.status.Offset)
	// Check the subscriber's notification
	notification, ok := <-notificationCh
	assert.Equal(dt.T(), true, ok)
	jobStatus := JobStatus{Name: Downloading, Err: nil, Offset: 10}
	assert.True(dt.T(), reflect.DeepEqual(jobStatus, notification))
}

func (dt *downloaderTest) Test_updateStatusOffset_InsertNew() {
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: dt.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	assert.Nil(dt.T(), err)
	value := dt.cache.Erase(fileInfoKeyName)
	assert.True(dt.T(), value != nil)

	err = dt.job.updateStatusOffset(10)

	assert.NotNil(dt.T(), err)
	assert.True(dt.T(), strings.Contains(err.Error(), lru.EntryNotExistErrMsg))
	// Confirm job's status offset
	assert.EqualValues(dt.T(), 0, dt.job.status.Offset)
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
	assert.Nil(dt.T(), err)
	_, err = dt.cache.Insert(fileInfoKeyName, fileInfo)
	assert.Nil(dt.T(), err)

	// Change the size of object and then try to update file info cache.
	dt.job.object.Size = 10
	err = dt.job.updateStatusOffset(15)

	assert.NotNil(dt.T(), err)
	assert.True(dt.T(), strings.Contains(err.Error(), lru.InvalidUpdateEntrySizeErrorMsg))
	// Confirm job's status offset
	assert.EqualValues(dt.T(), 0, dt.job.status.Offset)
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
	assert.True(dt.T(), errors.Is(cancelCtx.Err(), context.Canceled))
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	// doneCh is closed
	_, ok := <-dt.job.doneCh
	assert.False(dt.T(), ok)
	// References to context and cancel function are removed
	assert.Nil(dt.T(), dt.job.cancelCtx)
	assert.Nil(dt.T(), dt.job.cancelFunc)
	assert.Nil(dt.T(), dt.job.removeJobCallback)
	// Call back function should have been called
	assert.True(dt.T(), callbackExecuted.Load())
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
	assert.Nil(dt.T(), err)
	defer func() {
		_ = file.Close()
	}()

	// Start download
	err = dt.job.downloadObjectToFile(file)

	assert.Nil(dt.T(), err)
	jobStatus, ok := <-notificationC
	assert.Equal(dt.T(), true, ok)
	// Check the notification is sent after subscribed offset
	assert.GreaterOrEqual(dt.T(), jobStatus.Offset, subscribedOffset)
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
	assert.Nil(dt.T(), err)
	defer func() {
		_ = file.Close()
	}()

	dt.job.cancelFunc()
	err = dt.job.downloadObjectToFile(file)

	assert.True(dt.T(), errors.Is(err, context.Canceled), fmt.Sprintf("didn't get context canceled error: %v", err))
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
	assert.True(dt.T(), reflect.DeepEqual(jobStatus, dt.job.status))
	// Verify file is downloaded
	dt.verifyFile(objectContent)
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(objectSize))
	// Verify callback is executed and removed
	assert.True(dt.T(), callbackExecuted.Load())
	assert.Nil(dt.T(), dt.job.removeJobCallback)
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
	assert.True(dt.T(), reflect.DeepEqual(jobStatus, dt.job.status))
	// Verify file is downloaded
	dt.verifyFile(objectContent)
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(objectSize))
	// Verify callback is executed and removed
	assert.True(dt.T(), callbackExecuted.Load())
	assert.Nil(dt.T(), dt.job.removeJobCallback)
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
	assert.GreaterOrEqual(dt.T(), jobStatus.Offset, subscribedOffset)
	// check job completed successfully
	jobStatus = JobStatus{Completed, nil, int64(objectSize)}
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	assert.True(dt.T(), reflect.DeepEqual(jobStatus, dt.job.status))
	// verify file is downloaded
	dt.verifyFile(objectContent)
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(objectSize))
	// Verify callback is executed and removed
	assert.True(dt.T(), callbackExecuted.Load())
	assert.Nil(dt.T(), dt.job.removeJobCallback)
}

func (dt *downloaderTest) Test_Download_WhenNotStarted() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 16 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})

	// Start download
	offset := int64(8 * util.MiB)
	jobStatus, err := dt.job.Download(context.Background(), offset, true)

	assert.Nil(dt.T(), err)
	// Verify that jobStatus is downloading and downloaded more than 8 Mib.
	assert.Equal(dt.T(), Downloading, jobStatus.Name)
	assert.Nil(dt.T(), jobStatus.Err)
	assert.GreaterOrEqual(dt.T(), jobStatus.Offset, offset)
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
	assert.Nil(dt.T(), err)
	assert.Equal(dt.T(), Downloading, jobStatus.Name)

	// Again call download but wait for download this time.
	offset := int64(25 * util.MiB)
	jobStatus, err = dt.job.Download(ctx, offset, true)

	assert.Nil(dt.T(), err)
	// Verify that jobStatus is downloading and downloaded at least 25 Mib.
	assert.Equal(dt.T(), Downloading, jobStatus.Name)
	assert.Nil(dt.T(), jobStatus.Err)
	assert.GreaterOrEqual(dt.T(), jobStatus.Offset, offset)
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
	assert.Nil(dt.T(), err)
	// Verify that jobStatus is Downloading but offset is object size
	expectedJobStatus := JobStatus{Downloading, nil, int64(objectSize)}
	assert.True(dt.T(), reflect.DeepEqual(expectedJobStatus, jobStatus))
	dt.waitForCrcCheckToBeCompleted()
	assert.Equal(dt.T(), Completed, dt.job.status.Name)

	// Try to request for some offset when job was already completed.
	offset := int64(16 * util.MiB)
	jobStatus, err = dt.job.Download(ctx, offset, true)

	assert.Nil(dt.T(), err)
	// Verify that jobStatus is completed & offset returned is still 16 MiB
	// this ensures that async job is not started again.
	assert.Equal(dt.T(), Completed, jobStatus.Name)
	assert.Nil(dt.T(), jobStatus.Err)
	assert.GreaterOrEqual(dt.T(), jobStatus.Offset, int64(objectSize))
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

	assert.Nil(dt.T(), err)
	// Verify that jobStatus is failed
	assert.Equal(dt.T(), Failed, jobStatus.Name)
	assert.GreaterOrEqual(dt.T(), int64(0), jobStatus.Offset)
	assert.True(dt.T(), strings.Contains(jobStatus.Err.Error(), lru.InvalidUpdateEntrySizeErrorMsg))
	// Verify callback is executed
	assert.True(dt.T(), callbackExecuted.Load())
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

	assert.Nil(dt.T(), err)
	assert.EqualValues(dt.T(), Failed, jobStatus.Name)
	assert.True(dt.T(), strings.Contains(jobStatus.Err.Error(), lru.InvalidUpdateEntrySizeErrorMsg))
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

	assert.Nil(dt.T(), err)
	assert.Equal(dt.T(), Invalid, jobStatus.Name)
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

	assert.NotNil(dt.T(), err)
	assert.True(dt.T(), strings.Contains(err.Error(), fmt.Sprintf("Download: the requested offset %d is greater than the size of object %d", offset, dt.object.Size)))
	expectedJobStatus := JobStatus{NotStarted, nil, 0}
	assert.True(dt.T(), reflect.DeepEqual(expectedJobStatus, jobStatus))
	assert.False(dt.T(), callbackExecuted.Load())
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

	assert.NotNil(dt.T(), err)
	assert.True(dt.T(), errors.Is(err, context.DeadlineExceeded))
	// jobStatus is empty in this case.
	assert.EqualValues(dt.T(), "", jobStatus.Name)
	assert.Nil(dt.T(), jobStatus.Err)
	// job should be either Downloading or Completed.
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	assert.True(dt.T(), dt.job.status.Name == Downloading || dt.job.status.Name == Completed)
	if dt.job.status.Name == Downloading {
		assert.False(dt.T(), callbackExecuted.Load())
	} else {
		assert.True(dt.T(), callbackExecuted.Load())
	}
	assert.Nil(dt.T(), dt.job.status.Err)
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
		assert.NotEqualValues(dt.T(), Failed, jobStatus.Name)
		if expectedErr != nil {
			assert.True(dt.T(), strings.Contains(err.Error(), expectedErr.Error()))
			return
		} else {
			assert.Equal(dt.T(), expectedErr, err)
		}
		assert.GreaterOrEqual(dt.T(), jobStatus.Offset, expectedOffset)
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
	assert.True(dt.T(), reflect.DeepEqual(expectedJobStatus, dt.job.status))
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
	assert.Nil(dt.T(), err)
	assert.Equal(dt.T(), Downloading, jobStatus.Name)

	// GetStatus in between downloading
	jobStatus = dt.job.GetStatus()

	assert.True(dt.T(), (jobStatus.Name == Downloading) || (jobStatus.Name == Completed))
	assert.Nil(dt.T(), jobStatus.Err)
	assert.LessOrEqual(dt.T(), int64(0), jobStatus.Offset)
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
	assert.Nil(dt.T(), err)
	assert.Equal(dt.T(), Downloading, jobStatus.Name)

	dt.job.Invalidate()

	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	dt.verifyInvalidError(dt.job.status.Err)
	assert.Equal(dt.T(), Invalid, dt.job.status.Name)
	assert.True(dt.T(), callbackExecuted.Load())
	assert.Nil(dt.T(), dt.job.removeJobCallback)
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
	assert.Nil(dt.T(), dt.job.status.Err)
	assert.Equal(dt.T(), Invalid, dt.job.status.Name)
	assert.True(dt.T(), callbackExecuted.Load())
	assert.Nil(dt.T(), dt.job.removeJobCallback)
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
	assert.Nil(dt.T(), err)
	dt.waitForCrcCheckToBeCompleted()
	jobStatus := dt.job.GetStatus()
	assert.Equal(dt.T(), Completed, jobStatus.Name)

	dt.job.Invalidate()

	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	assert.Equal(dt.T(), Invalid, dt.job.status.Name)
	dt.verifyInvalidError(dt.job.status.Err)
	assert.EqualValues(dt.T(), 1, callbackExecutionCount.Load())
	assert.Nil(dt.T(), dt.job.removeJobCallback)
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
	assert.Nil(dt.T(), err)
	assert.Equal(dt.T(), Downloading, jobStatus.Name)
	wg := sync.WaitGroup{}
	invalidateFunc := func() {
		defer wg.Done()
		dt.job.Invalidate()
		currJobStatus := dt.job.GetStatus()
		assert.Equal(dt.T(), Invalid, currJobStatus.Name)
		dt.verifyInvalidError(currJobStatus.Err)
	}

	// start concurrent Invalidate
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go invalidateFunc()
	}
	wg.Wait()

	assert.EqualValues(dt.T(), 1, callbackExecutionCount.Load())
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	assert.Nil(dt.T(), dt.job.removeJobCallback)
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
		assert.Nil(dt.T(), err)
		assert.True(dt.T(), jobStatus.Name == Downloading || jobStatus.Name == Invalid || jobStatus.Name == Completed)
		// If status is downloading/complete and wait for download is true then
		// status offset should be at least requested offset.
		if waitForDownload && (jobStatus.Name == Downloading || jobStatus.Name == Completed) {
			assert.GreaterOrEqual(dt.T(), jobStatus.Offset, offset)
		}
	}
	invalidateFunc := func() {
		defer wg.Done()
		dt.job.Invalidate()
		currJobStatus := dt.job.GetStatus()
		assert.Equal(dt.T(), Invalid, currJobStatus.Name)
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

	assert.EqualValues(dt.T(), 1, callbackExecutionCount.Load())
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	assert.Nil(dt.T(), dt.job.removeJobCallback)
}

func (dt *downloaderTest) Test_validateCRC_ForTamperedFileWhenEnableCRCIsTrue() {
	objectName := "path/in/gcs/file1.txt"
	objectSize := 8 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})
	// Start download
	offset := int64(8 * util.MiB)
	jobStatus, err := dt.job.Download(context.Background(), offset, true)
	assert.Nil(dt.T(), err)
	// Here the crc check will be successful
	dt.waitForCrcCheckToBeCompleted()
	assert.Equal(dt.T(), Completed, dt.job.status.Name)
	assert.Nil(dt.T(), dt.job.status.Err)
	assert.GreaterOrEqual(dt.T(), dt.job.status.Offset, offset)
	// Verify file
	dt.verifyFile(objectContent[:jobStatus.Offset])
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(jobStatus.Offset))
	// Tamper the file
	err = os.WriteFile(dt.fileSpec.Path, []byte("test"), 0644)
	assert.Nil(dt.T(), err)

	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())
	err = dt.job.validateCRC()

	assert.NotNil(dt.T(), err)
	assert.True(dt.T(), strings.Contains(err.Error(), "checksum mismatch detected"))
	assert.Nil(dt.T(), dt.getFileInfo())
	_, err = os.Stat(dt.fileSpec.Path)
	assert.NotNil(dt.T(), err)
	assert.True(dt.T(), strings.Contains(err.Error(), "no such file or directory"))
}

func (dt *downloaderTest) Test_validateCRC_ForTamperedFileWhenEnableCRCIsFalse() {
	objectName := "path/in/gcs/file2.txt"
	objectSize := 1 * util.MiB
	objectContent := testutil.GenerateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize), func() {})
	// Start download
	offset := int64(1 * util.MiB)
	jobStatus, err := dt.job.Download(context.Background(), offset, true)
	assert.Nil(dt.T(), err)
	// Here the crc check will be successful
	dt.waitForCrcCheckToBeCompleted()
	assert.Equal(dt.T(), Completed, dt.job.status.Name)
	assert.Nil(dt.T(), dt.job.status.Err)
	assert.GreaterOrEqual(dt.T(), dt.job.status.Offset, offset)
	// Verify file
	dt.verifyFile(objectContent[:jobStatus.Offset])
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(jobStatus.Offset))
	// Tamper the file
	err = os.WriteFile(dt.fileSpec.Path, []byte("test"), 0644)
	assert.Nil(dt.T(), err)
	dt.job.fileCacheConfig.EnableCrc = false

	dt.job.cancelCtx, dt.job.cancelFunc = context.WithCancel(context.Background())
	err = dt.job.validateCRC()

	assert.Nil(dt.T(), err)
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
	assert.Nil(dt.T(), err)
	assert.True(dt.T(), (dt.job.status.Name == Downloading) || (dt.job.status.Name == Completed), fmt.Sprintf("got job status: %v", dt.job.status.Name))
	assert.Nil(dt.T(), dt.job.status.Err)
	assert.GreaterOrEqual(dt.T(), dt.job.status.Offset, offset)

	dt.job.cancelFunc()
	dt.waitForCrcCheckToBeCompleted()

	assert.Equal(dt.T(), Invalid, dt.job.status.Name)
	dt.verifyInvalidError(dt.job.status.Err)
}

func (dt *downloaderTest) Test_handleError_SetStatusAsInvalidWhenContextIsCancelled() {
	subscriberOffset := int64(1)
	notificationC := dt.job.subscribe(subscriberOffset)
	err := errors.Join(context.Canceled)

	err = fmt.Errorf("Wrapping with custom message %w", err)
	dt.job.handleError(err)

	assert.Equal(dt.T(), 0, dt.job.subscribers.Len())
	notification, ok := <-notificationC
	assert.Equal(dt.T(), Invalid, notification.Name)
	dt.verifyInvalidError(notification.Err)
	assert.EqualValues(dt.T(), 0, notification.Offset)
	assert.True(dt.T(), ok)
}

func (dt *downloaderTest) Test_handleError_SetStatusAsErrorWhenContextIsNotCancelled() {
	subscriberOffset := int64(1)
	notificationC := dt.job.subscribe(subscriberOffset)
	err := errors.New("custom error")

	updatedErr := fmt.Errorf("Custom message %w", err)
	dt.job.handleError(updatedErr)

	assert.Equal(dt.T(), 0, dt.job.subscribers.Len())
	notification, ok := <-notificationC
	jobStatus := JobStatus{Name: Failed, Err: updatedErr, Offset: 0}
	fmt.Println(notification)
	assert.True(dt.T(), reflect.DeepEqual(jobStatus, notification))
	assert.Equal(dt.T(), true, ok)
}

func (dt *downloaderTest) Test_When_Parallel_Download_Is_Enabled() {
	//Arrange - initJobTest is being called in setup of downloader.go
	dt.job.fileCacheConfig.EnableParallelDownloads = true

	result := dt.job.IsParallelDownloadsEnabled()

	assert.True(dt.T(), result)
}

func (dt *downloaderTest) Test_When_Parallel_Download_Is_Disabled() {
	//Arrange - initJobTest is being called in setup of downloader.go
	dt.job.fileCacheConfig.EnableParallelDownloads = false

	result := dt.job.IsParallelDownloadsEnabled()

	assert.False(dt.T(), result)
}

func (dt *downloaderTest) Test_createCacheFile_WhenNonParallelDownloads() {
	//Arrange - initJobTest is being called in setup of downloader.go
	dt.job.fileCacheConfig.EnableParallelDownloads = false

	cacheFile, err := dt.job.createCacheFile()

	assert.Nil(dt.T(), err)
	defer func() {
		_ = cacheFile.Close()
	}()
}

func (dt *downloaderTest) Test_createCacheFile_WhenParallelDownloads() {
	//Arrange - initJobTest is being called in setup of downloader.go
	dt.job.fileCacheConfig.EnableParallelDownloads = true

	cacheFile, err := dt.job.createCacheFile()

	assert.Nil(dt.T(), err)
	defer func() {
		_ = cacheFile.Close()
	}()
}

func (dt *downloaderTest) Test_createCacheFile_WhenParallelDownloadsEnabledAndODirectDisabled() {
	//Arrange - initJobTest is being called in setup of downloader.go
	dt.job.fileCacheConfig.EnableParallelDownloads = true
	dt.job.fileCacheConfig.EnableODirect = false

	cacheFile, err := dt.job.createCacheFile()

	assert.Nil(dt.T(), err)
	defer func() {
		_ = cacheFile.Close()
	}()
}
