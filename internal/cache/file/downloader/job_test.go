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
	"fmt"
	"math/rand"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
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
	object, err := dt.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: objectName,
		ForceFetchFromGcs: true})
	if err != nil {
		panic(fmt.Errorf("error whlie stating object: %v", err))
	}
	minObject := gcs.MinObject{
		Name:            object.Name,
		Size:            object.Size,
		Generation:      object.Generation,
		MetaGeneration:  object.MetaGeneration,
		Updated:         object.Updated,
		Metadata:        object.Metadata,
		ContentEncoding: object.ContentEncoding,
	}
	return minObject
}

func (dt *downloaderTest) initJobTest(objectName string, objectContent []byte, sequentialReadSize int32, lruCacheSize uint64) {
	ctx := context.Background()
	objects := map[string][]byte{objectName: objectContent}
	err := storageutil.CreateObjects(ctx, dt.bucket, objects)
	AssertEq(nil, err)
	dt.object = dt.getMinObject(objectName)
	dt.fileSpec = data.FileSpec{Path: dt.fileCachePath(dt.bucket.Name(), dt.object.Name), Perm: util.DefaultFilePerm}
	dt.cache = lru.NewCache(lruCacheSize)
	dt.job = NewJob(&dt.object, dt.bucket, dt.cache, sequentialReadSize, dt.fileSpec)
}

func (dt *downloaderTest) verifyFile(content []byte) {
	fileStat, err := os.Stat(dt.fileSpec.Path)
	AssertEq(nil, err)
	AssertEq(dt.fileSpec.Perm, fileStat.Mode())
	AssertEq(len(content), fileStat.Size())
	// verify the content of file downloaded. only verified till
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
	AssertEq(offset, fileInfo.(data.FileInfo).Offset)
	AssertEq(dt.object.Size, fileInfo.(data.FileInfo).Size())
}

func (dt *downloaderTest) fileCachePath(bucketName string, objectName string) string {
	return path.Join(cacheLocation, bucketName, objectName)
}

func generateRandomBytes(length int) []byte {
	randBytes := make([]byte, length)
	for i := 0; i < length; i++ {
		randBytes[i] = byte(rand.Intn(26) + 65)
	}
	return randBytes
}

func (dt *downloaderTest) Test_init() {
	// Explicitly changing initialized values first.
	dt.job.status.Name = DOWNLOADING
	dt.job.status.Err = fmt.Errorf("some error")
	dt.job.status.Offset = -1
	dt.job.subscribers.PushBack(struct{}{})
	dt.job.cancelCtx = nil
	dt.job.cancelFunc = nil

	dt.job.init()

	AssertEq(NOT_STARTED, dt.job.status.Name)
	AssertEq(nil, dt.job.status.Err)
	AssertEq(0, dt.job.status.Offset)
	AssertTrue(reflect.DeepEqual(list.List{}, dt.job.subscribers))
	AssertNe(nil, dt.job.cancelCtx)
	AssertNe(nil, dt.job.cancelFunc)
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
	// check 1st and 2nd subscribers
	var subscriber jobSubscriber
	AssertEq(reflect.TypeOf(subscriber), reflect.TypeOf(dt.job.subscribers.Front().Value.(jobSubscriber)))
	AssertEq(0, dt.job.subscribers.Front().Value.(jobSubscriber).subscribedOffset)
	AssertEq(reflect.TypeOf(subscriber), reflect.TypeOf(dt.job.subscribers.Back().Value.(jobSubscriber)))
	AssertEq(1, dt.job.subscribers.Back().Value.(jobSubscriber).subscribedOffset)
}

func (dt *downloaderTest) Test_notifySubscriber_FAILED() {
	subscriberOffset := int64(1)
	notificationC := dt.job.subscribe(subscriberOffset)
	dt.job.status.Name = FAILED
	customErr := fmt.Errorf("custom err")
	dt.job.status.Err = customErr

	dt.job.notifySubscribers()

	AssertEq(0, dt.job.subscribers.Len())
	notification, ok := <-notificationC
	jobStatus := JobStatus{Name: FAILED, Err: customErr, Offset: 0}
	AssertTrue(reflect.DeepEqual(jobStatus, notification))
	AssertEq(true, ok)
}

func (dt *downloaderTest) Test_notifySubscriber_CANCELLED() {
	subscriberOffset := int64(1)
	notificationC := dt.job.subscribe(subscriberOffset)
	dt.job.status.Name = CANCELLED

	dt.job.notifySubscribers()

	AssertEq(0, dt.job.subscribers.Len())
	notification, ok := <-notificationC
	jobStatus := JobStatus{Name: CANCELLED, Err: nil, Offset: 0}
	AssertTrue(reflect.DeepEqual(jobStatus, notification))
	AssertEq(true, ok)
}

func (dt *downloaderTest) Test_notifySubscriber_SubscribedOffset() {
	subscriberOffset1 := int64(3)
	subscriberOffset2 := int64(5)
	notificationC1 := dt.job.subscribe(subscriberOffset1)
	_ = dt.job.subscribe(subscriberOffset2)
	dt.job.status.Name = DOWNLOADING
	dt.job.status.Offset = 4

	dt.job.notifySubscribers()

	AssertEq(1, dt.job.subscribers.Len())
	notification1, ok := <-notificationC1
	jobStatus := JobStatus{Name: DOWNLOADING, Err: nil, Offset: 4}
	AssertTrue(reflect.DeepEqual(jobStatus, notification1))
	AssertEq(true, ok)
	// check 2nd subscriber's offset
	AssertEq(subscriberOffset2, dt.job.subscribers.Front().Value.(jobSubscriber).subscribedOffset)
}

func (dt *downloaderTest) Test_failWhileDownloading() {
	subscriberOffset1 := int64(3)
	subscriberOffset2 := int64(5)
	notificationC1 := dt.job.subscribe(subscriberOffset1)
	notificationC2 := dt.job.subscribe(subscriberOffset2)
	dt.job.status = JobStatus{Name: DOWNLOADING, Err: nil, Offset: 4}

	customErr := fmt.Errorf("custom error")
	dt.job.failWhileDownloading(customErr)

	AssertEq(0, dt.job.subscribers.Len())
	notification1, ok1 := <-notificationC1
	notification2, ok2 := <-notificationC2
	jobStatus := JobStatus{Name: FAILED, Err: customErr, Offset: 4}
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
	// confirm fileInfoCache is updated with new offset.
	lookupResult := dt.cache.LookUp(fileInfoKeyName)
	AssertFalse(lookupResult == nil)
	fileInfo = lookupResult.(data.FileInfo)
	AssertEq(1, fileInfo.Offset)
	AssertEq(dt.job.object.Generation, fileInfo.ObjectGeneration)
	AssertEq(dt.job.object.Size, fileInfo.FileSize)
}

// This test should fail when we shift to only updating fileInfoCache in Job.
// This test should be removed when that happens.
func (dt *downloaderTest) Test_updateFileInfoCache_InsertNew() {
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: DefaultObjectName,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	dt.job.status.Offset = 1

	err = dt.job.updateFileInfoCache()

	AssertEq(nil, err)
	// confirm fileInfoCache is updated with new offset.
	lookupResult := dt.cache.LookUp(fileInfoKeyName)
	AssertFalse(lookupResult == nil)
	fileInfo := lookupResult.(data.FileInfo)
	AssertEq(1, fileInfo.Offset)
	AssertEq(dt.job.object.Generation, fileInfo.ObjectGeneration)
	AssertEq(dt.job.object.Size, fileInfo.FileSize)
}

func (dt *downloaderTest) Test_updateFileInfoCache_Fail() {
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: DefaultObjectName,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	// set size of object more than MaxSize of cache.
	dt.job.object.Size = 100

	err = dt.job.updateFileInfoCache()

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), lru.InvalidEntrySizeErrorMsg))
	// confirm fileInfoCache is not updated.
	lookupResult := dt.cache.LookUp(fileInfoKeyName)
	AssertTrue(lookupResult == nil)
}

// Note: We can't test Test_downloadObjectAsync_MoreThanSequentialReadSize as
// the fake storage bucket/server in the testing environment doesn't support
// reading ranges (start and limit in NewReader call)
func (dt *downloaderTest) Test_downloadObjectAsync_LessThanSequentialReadSize() {
	// Create new object in bucket and create new job for it.
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize))

	// start download
	dt.job.downloadObjectAsync()

	// check job completed successfully
	jobStatus := JobStatus{COMPLETED, nil, int64(objectSize)}
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertTrue(reflect.DeepEqual(jobStatus, dt.job.status))
	// verify file is downloaded
	dt.verifyFile(objectContent)
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(objectSize))
}

func (dt *downloaderTest) Test_downloadObjectAsync_LessThanChunkSize() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 2 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, 25, uint64(2*objectSize))

	// start download
	dt.job.downloadObjectAsync()

	// check job completed successfully
	jobStatus := JobStatus{COMPLETED, nil, int64(objectSize)}
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertTrue(reflect.DeepEqual(jobStatus, dt.job.status))
	// verify file is downloaded
	dt.verifyFile(objectContent)
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(objectSize))
}

func (dt *downloaderTest) Test_downloadObjectAsync_Notification() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize))
	// Add subscriber
	subscribedOffset := int64(45 * util.MiB)
	notificationC := dt.job.subscribe(subscribedOffset)

	// start download
	dt.job.downloadObjectAsync()

	jobStatus := <-notificationC
	// check the notification is sent after subscribed offset
	AssertGe(jobStatus.Offset, subscribedOffset)
	// check job completed successfully
	jobStatus = JobStatus{COMPLETED, nil, int64(objectSize)}
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertTrue(reflect.DeepEqual(jobStatus, dt.job.status))
	// verify file is downloaded
	dt.verifyFile(objectContent)
	// Verify fileInfoCache update
	dt.verifyFileInfoEntry(uint64(objectSize))
}

func (dt *downloaderTest) Test_downloadObjectAsync_ErrorWhenFileCacheHasLessSize() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize-1))

	// start download
	dt.job.downloadObjectAsync()

	// check job failed
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	AssertEq(FAILED, dt.job.status.Name)
	AssertTrue(strings.Contains(dt.job.status.Err.Error(), "size of the entry is more than the cache's maxSize"))
}

func (dt *downloaderTest) Test_Download_WhenNotStarted() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize))

	// start download
	offset := int64(25 * util.MiB)
	jobStatus, err := dt.job.Download(context.Background(), offset, true)

	AssertEq(nil, err)
	// verify that jobStatus is downloading and downloaded more than 25 Mib.
	AssertEq(DOWNLOADING, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, offset)
	// verify that after some time the whole object is downloaded
	time.Sleep(time.Second * 2)
	// verify file
	dt.verifyFile(objectContent)
	// verify file info cache
	dt.verifyFileInfoEntry(uint64(objectSize))
}

func (dt *downloaderTest) Test_Download_WhenAlreadyDownloading() {
	// Create new object in bucket and create new job for it.
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize))
	// start download but not wait for download
	ctx := context.Background()
	jobStatus, err := dt.job.Download(ctx, 1, false)
	AssertEq(nil, err)
	AssertEq(DOWNLOADING, jobStatus.Name)

	// Again call download but wait for download this time.
	offset := int64(25 * util.MiB)
	jobStatus, err = dt.job.Download(ctx, offset, true)

	AssertEq(nil, err)
	// verify that jobStatus is downloading and downloaded at least 25 Mib.
	AssertEq(DOWNLOADING, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, offset)
	// verify that after some time the whole object is downloaded
	time.Sleep(time.Second * 2)
	// verify file
	dt.verifyFile(objectContent)
	// verify file info cache
	dt.verifyFileInfoEntry(uint64(objectSize))
}

func (dt *downloaderTest) Test_Download_WhenAlreadyCompleted() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize))
	// Wait for whole download to be completed.
	ctx := context.Background()
	jobStatus, err := dt.job.Download(ctx, int64(objectSize), true)
	AssertEq(nil, err)
	// verify that jobStatus is DOWNLOADING but offset is object size
	expectedJobStatus := JobStatus{DOWNLOADING, nil, int64(objectSize)}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))

	// try to request for some offset when job was already completed.
	offset := int64(18 * util.MiB)
	jobStatus, err = dt.job.Download(ctx, offset, true)

	AssertEq(nil, err)
	// verify that jobStatus is completed & offset returned is still 25 MiB
	// this ensures that async job is not started again.
	AssertEq(COMPLETED, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, objectSize)
	// verify file
	dt.verifyFile(objectContent)
	// verify file info cache
	dt.verifyFileInfoEntry(uint64(jobStatus.Offset))
}

func (dt *downloaderTest) Test_Download_WhenAsyncFails() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	// set size of cache smaller than object size.
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize-1))

	// Wait for whole download to be completed/failed.
	ctx := context.Background()
	jobStatus, err := dt.job.Download(ctx, int64(objectSize), true)

	AssertEq(nil, err)
	// verify that jobStatus is failed
	AssertEq(FAILED, jobStatus.Name)
	AssertEq(ReadChunkSize, jobStatus.Offset)
	AssertTrue(strings.Contains(jobStatus.Err.Error(), "size of the entry is more than the cache's maxSize"))
}

func (dt *downloaderTest) Test_Download_AlreadyFailed() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize-1))
	// Wait for whole download to be completed/failed.
	jobStatus, err := dt.job.Download(context.Background(), int64(objectSize), true)
	AssertEq(nil, err)
	// verify that jobStatus is failed
	AssertEq(FAILED, jobStatus.Name)
	AssertEq(ReadChunkSize, jobStatus.Offset)
	AssertTrue(strings.Contains(jobStatus.Err.Error(), "size of the entry is more than the cache's maxSize"))

	// requesting again from download job which is in failed state
	jobStatus, err = dt.job.Download(context.Background(), int64(objectSize), true)

	AssertEq(nil, err)
	AssertEq(FAILED, jobStatus.Name)
	AssertEq(ReadChunkSize, jobStatus.Offset)
	AssertTrue(strings.Contains(jobStatus.Err.Error(), "size of the entry is more than the cache's maxSize"))
}

func (dt *downloaderTest) Test_Download_AlreadyInvalid() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 1 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize-1))
	// make the state as invalid
	dt.job.mu.Lock()
	dt.job.status.Name = INVALID
	dt.job.mu.Unlock()

	// requesting download when already invalid.
	jobStatus, err := dt.job.Download(context.Background(), int64(objectSize), true)

	AssertEq(nil, err)
	AssertEq(INVALID, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
}

func (dt *downloaderTest) Test_Download_InvalidOffset() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize))

	// requesting invalid offset
	offset := int64(objectSize) + 1
	jobStatus, err := dt.job.Download(context.Background(), offset, true)

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), fmt.Sprintf("Download: the requested offset %d is greater than the size of object %d", offset, dt.object.Size)))
	expectedJobStatus := JobStatus{NOT_STARTED, nil, 0}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))
}

func (dt *downloaderTest) Test_Download_CtxCancelled() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))

	// requesting full download and then the download call should be cancelled after
	// timeout but async download shouldn't be cancelled
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Millisecond*2)
	defer cancelFunc()
	offset := int64(objectSize)
	jobStatus, err := dt.job.Download(ctx, offset, true)

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "context deadline exceeded"))
	// jobStatus is empty in this case.
	AssertEq("", jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	// job should be completed after sometime as the timeout is on Download call
	// and not async download
	time.Sleep(time.Second * 2)
	jobStatus, err = dt.job.Download(context.Background(), 0, false)
	AssertEq(nil, err)
	expectedJobStatus := JobStatus{COMPLETED, nil, int64(objectSize)}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))
	// verify file
	dt.verifyFile(objectContent)
	// verify file info cache
	dt.verifyFileInfoEntry(uint64(objectSize))
}

func (dt *downloaderTest) Test_Download_Concurrent() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	ctx := context.Background()
	wg := sync.WaitGroup{}
	offsets := []int64{0, 4 * util.MiB, 16 * util.MiB, 8 * util.MiB, int64(objectSize), int64(objectSize) + 1}
	expectedErrs := []error{nil, nil, nil, nil, nil, fmt.Errorf(fmt.Sprintf("Download: the requested offset %d is greater than the size of object %d", int64(objectSize)+1, int64(objectSize)))}
	downloadFunc := func(expectedOffset int64, expectedErr error) {
		defer wg.Done()
		var jobStatus JobStatus
		var err error
		jobStatus, err = dt.job.Download(ctx, expectedOffset, true)
		AssertNe(FAILED, jobStatus.Name)
		if expectedErr != nil {
			AssertTrue(strings.Contains(err.Error(), expectedErr.Error()))
			return
		} else {
			AssertEq(expectedErr, err)
		}
		AssertGe(jobStatus.Offset, expectedOffset)
	}

	// start concurrent downloads
	for i, offset := range offsets {
		wg.Add(1)
		go downloadFunc(offset, expectedErrs[i])
	}
	wg.Wait()

	jobStatus, err := dt.job.Download(context.Background(), 0, false)
	AssertEq(nil, err)
	expectedJobStatus := JobStatus{COMPLETED, nil, int64(objectSize)}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))
	// verify file
	dt.verifyFile(objectContent)
	// verify file info cache
	dt.verifyFileInfoEntry(uint64(objectSize))
}

func (dt *downloaderTest) Test_Cancel_WhenDownlooading() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	// request for 2 MiB download to start downloading
	offset := int64(2 * util.MiB)
	jobStatus, err := dt.job.Download(context.Background(), offset, true)
	AssertEq(nil, err)
	AssertEq(DOWNLOADING, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, offset)

	// Wait for some time and then cancel
	time.Sleep(time.Millisecond * 30)
	dt.job.Cancel()

	jobStatus, err = dt.job.Download(context.Background(), 0, false)
	AssertEq(nil, err)
	AssertEq(CANCELLED, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	// file is not completely downloaded as job was still running when cancelled.
	AssertLt(jobStatus.Offset, objectSize)
	// job should not be completed even after sometime.
	time.Sleep(time.Second)
	newJobStatus, err := dt.job.Download(context.Background(), 0, false)
	AssertEq(nil, err)
	AssertEq(CANCELLED, newJobStatus.Name)
	AssertEq(nil, newJobStatus.Err)
	// file is not completely downloaded as job was still running when cancelled.
	AssertLt(newJobStatus.Offset, objectSize)
	// verify file downloaded till the offset
	dt.verifyFile(objectContent[:newJobStatus.Offset])
}

func (dt *downloaderTest) Test_Cancel_WhenAlreadyCompleted() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	// download complete object
	jobStatus, err := dt.job.Download(context.Background(), int64(objectSize), true)
	AssertEq(nil, err)
	expectedJobStatus := JobStatus{DOWNLOADING, nil, int64(objectSize)}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))

	dt.job.Cancel()

	// status is not changed to Cancelled
	jobStatus, err = dt.job.Download(context.Background(), 0, false)
	AssertEq(nil, err)
	AssertNe(CANCELLED, jobStatus.Name)
	AssertEq(nil, err)
	AssertEq(objectSize, jobStatus.Offset)
	// verify file downloaded till the offset
	dt.verifyFile(objectContent)
	// verify file info cache
	dt.verifyFileInfoEntry(uint64(objectSize))
}

func (dt *downloaderTest) Test_Cancel_WhenNotStarted() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))

	dt.job.Cancel()

	// status is changed to Cancelled
	expectedJobStatus := JobStatus{CANCELLED, nil, 0}
	jobStatus, err := dt.job.Download(context.Background(), 0, false)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))
	// verify file is not created
	_, err = os.Stat(dt.fileSpec.Path)
	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

func (dt *downloaderTest) Test_Cancel_Concurrent() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	ctx := context.Background()
	// start download without waiting
	jobStatus, err := dt.job.Download(ctx, 0, false)
	AssertEq(nil, err)
	AssertEq(DOWNLOADING, jobStatus.Name)
	// wait for sometime to allow downloading before cancelling
	time.Sleep(time.Millisecond * 10)
	wg := sync.WaitGroup{}
	cancelFunc := func() {
		defer wg.Done()
		dt.job.Cancel()
		currJobStatus, currErr := dt.job.Download(ctx, 1, true)
		AssertEq(CANCELLED, currJobStatus.Name)
		AssertEq(nil, currErr)
		AssertGe(currJobStatus.Offset, 0)
	}

	// start concurrent cancel
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go cancelFunc()
	}
	wg.Wait()

	jobStatus, err = dt.job.Download(ctx, 1, true)
	AssertEq(nil, err)
	AssertEq(CANCELLED, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	// verify file
	dt.verifyFile(objectContent[:jobStatus.Offset])
}

func (dt *downloaderTest) Test_GetStatus() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 20 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	ctx := context.Background()
	// start download without waiting
	jobStatus, err := dt.job.Download(ctx, 0, false)
	AssertEq(nil, err)
	AssertEq(DOWNLOADING, jobStatus.Name)
	// wait for sometime to get some data downloaded
	time.Sleep(10 * time.Millisecond)

	// GetStatus in between downloading
	jobStatus = dt.job.GetStatus()

	AssertEq(DOWNLOADING, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, 0)
	// verify file
	dt.verifyFile(objectContent[:jobStatus.Offset])
}

func (dt *downloaderTest) Test_Invalidate_WhenDownloading() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	ctx := context.Background()
	// start download without waiting
	jobStatus, err := dt.job.Download(ctx, 0, false)
	AssertEq(nil, err)
	AssertEq(DOWNLOADING, jobStatus.Name)

	dt.job.Invalidate()

	jobStatus = dt.job.GetStatus()
	AssertEq(nil, jobStatus.Err)
	AssertEq(INVALID, jobStatus.Name)
}

func (dt *downloaderTest) Test_Invalidate_Concurrent() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 30 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	dt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	ctx := context.Background()
	// start download without waiting
	jobStatus, err := dt.job.Download(ctx, 0, false)
	AssertEq(nil, err)
	AssertEq(DOWNLOADING, jobStatus.Name)
	wg := sync.WaitGroup{}
	invalidateFunc := func() {
		defer wg.Done()
		dt.job.Invalidate()
		currJobStatus := dt.job.GetStatus()
		AssertEq(INVALID, currJobStatus.Name)
		AssertEq(nil, currJobStatus.Err)
	}

	// start concurrent Invalidate
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go invalidateFunc()
	}
	wg.Wait()
}
