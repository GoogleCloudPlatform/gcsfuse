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
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	. "github.com/jacobsa/ogletest"
)

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const CacheMaxSize = 50
const DefaultObjectName = "foo"
const DefaultSequentialReadSizeMb = 100

var cacheLocation string = path.Join(os.Getenv("HOME"), "cache/location")

func TestJob(t *testing.T) { RunTests(t) }

type jobTest struct {
	job         *Job
	bucket      gcs.Bucket
	object      gcs.MinObject
	cache       *lru.Cache
	fakeStorage storage.FakeStorage
	fileSpec    data.FileSpec
}

func init() { RegisterTestSuite(&jobTest{}) }

func (jt *jobTest) SetUp(*TestInfo) {
	locker.EnableInvariantsCheck()
	// Create bucket in fake storage.
	jt.fakeStorage = storage.NewFakeStorage()
	storageHandle := jt.fakeStorage.CreateStorageHandle()
	jt.bucket = storageHandle.BucketHandle(storage.TestBucketName, "")

	jt.initJobTest(DefaultObjectName, []byte("taco"), 200, CacheMaxSize)
	operations.RemoveDir(cacheLocation)
}

func (jt *jobTest) TearDown() {
	jt.fakeStorage.ShutDown()
	operations.RemoveDir(cacheLocation)
}

func (jt *jobTest) getMinObject(objectName string) gcs.MinObject {
	ctx := context.Background()
	object, err := jt.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: objectName,
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

func (jt *jobTest) initJobTest(objectName string, objectContent []byte, sequentialReadSize int32, lruCacheSize uint64) {
	ctx := context.Background()
	objects := map[string][]byte{objectName: objectContent}
	err := storageutil.CreateObjects(ctx, jt.bucket, objects)
	AssertEq(nil, err)
	jt.object = jt.getMinObject(objectName)
	jt.fileSpec = data.FileSpec{Path: jt.fileCachePath(jt.bucket.Name(), jt.object.Name), Perm: os.FileMode(0644)}
	jt.cache = lru.NewCache(lruCacheSize)
	jt.job = NewJob(&jt.object, jt.bucket, jt.cache, sequentialReadSize, jt.fileSpec)
}

func (jt *jobTest) verifyFile(content []byte) {
	fileStat, err := os.Stat(jt.fileSpec.Path)
	AssertEq(nil, err)
	AssertEq(jt.fileSpec.Perm, fileStat.Mode())
	AssertEq(len(content), fileStat.Size())
	// verify the content of file downloaded. only verified till
	fileContent, err := os.ReadFile(jt.fileSpec.Path)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(content, fileContent[:len(content)]))
}

func (jt *jobTest) verifyFileInfoEntry(offset uint64) {
	fileInfoKey := data.FileInfoKey{BucketName: jt.bucket.Name(), ObjectName: jt.object.Name}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	fileInfo := jt.cache.LookUp(fileInfoKeyName)
	AssertTrue(fileInfo != nil)
	AssertEq(jt.object.Generation, fileInfo.(data.FileInfo).ObjectGeneration)
	AssertEq(offset, fileInfo.(data.FileInfo).Offset)
	AssertEq(jt.object.Size, fileInfo.(data.FileInfo).Size())
}

func (jt *jobTest) fileCachePath(bucketName string, objectName string) string {
	return path.Join(cacheLocation, bucketName, objectName)
}

func generateRandomBytes(length int) []byte {
	randBytes := make([]byte, length)
	for i := 0; i < length; i++ {
		randBytes[i] = byte(rand.Intn(26) + 65)
	}
	return randBytes
}

func (jt *jobTest) Test_init() {
	// Explicitly changing initialized values first.
	jt.job.status.Name = DOWNLOADING
	jt.job.status.Err = fmt.Errorf("some error")
	jt.job.status.Offset = -1
	jt.job.subscribers.PushBack(struct{}{})
	jt.job.cancelCtx = nil
	jt.job.cancelFunc = nil

	jt.job.init()

	AssertEq(NOT_STARTED, jt.job.status.Name)
	AssertEq(nil, jt.job.status.Err)
	AssertEq(0, jt.job.status.Offset)
	AssertTrue(reflect.DeepEqual(list.List{}, jt.job.subscribers))
	AssertNe(nil, jt.job.cancelCtx)
	AssertNe(nil, jt.job.cancelFunc)
}

func (jt *jobTest) Test_subscribe() {
	subscriberOffset1 := int64(0)
	subscriberOffset2 := int64(1)

	notificationC1 := jt.job.subscribe(subscriberOffset1)
	notificationC2 := jt.job.subscribe(subscriberOffset2)

	AssertEq(2, jt.job.subscribers.Len())
	receivingC := make(<-chan JobStatus, 1)
	AssertEq(reflect.TypeOf(receivingC), reflect.TypeOf(notificationC1))
	AssertEq(reflect.TypeOf(receivingC), reflect.TypeOf(notificationC2))
	// check 1st and 2nd subscribers
	var subscriber jobSubscriber
	AssertEq(reflect.TypeOf(subscriber), reflect.TypeOf(jt.job.subscribers.Front().Value.(jobSubscriber)))
	AssertEq(0, jt.job.subscribers.Front().Value.(jobSubscriber).subscribedOffset)
	AssertEq(reflect.TypeOf(subscriber), reflect.TypeOf(jt.job.subscribers.Back().Value.(jobSubscriber)))
	AssertEq(1, jt.job.subscribers.Back().Value.(jobSubscriber).subscribedOffset)
}

func (jt *jobTest) Test_notifySubscriber_FAILED() {
	subscriberOffset := int64(1)
	notificationC := jt.job.subscribe(subscriberOffset)
	jt.job.status.Name = FAILED
	customErr := fmt.Errorf("custom err")
	jt.job.status.Err = customErr

	jt.job.notifySubscribers()

	AssertEq(0, jt.job.subscribers.Len())
	notification, ok := <-notificationC
	jobStatus := JobStatus{Name: FAILED, Err: customErr, Offset: 0}
	AssertTrue(reflect.DeepEqual(jobStatus, notification))
	AssertEq(true, ok)
}

func (jt *jobTest) Test_notifySubscriber_CANCELLED() {
	subscriberOffset := int64(1)
	notificationC := jt.job.subscribe(subscriberOffset)
	jt.job.status.Name = CANCELLED

	jt.job.notifySubscribers()

	AssertEq(0, jt.job.subscribers.Len())
	notification, ok := <-notificationC
	jobStatus := JobStatus{Name: CANCELLED, Err: nil, Offset: 0}
	AssertTrue(reflect.DeepEqual(jobStatus, notification))
	AssertEq(true, ok)
}

func (jt *jobTest) Test_notifySubscriber_SubscribedOffset() {
	subscriberOffset1 := int64(3)
	subscriberOffset2 := int64(5)
	notificationC1 := jt.job.subscribe(subscriberOffset1)
	_ = jt.job.subscribe(subscriberOffset2)
	jt.job.status.Name = DOWNLOADING
	jt.job.status.Offset = 4

	jt.job.notifySubscribers()

	AssertEq(1, jt.job.subscribers.Len())
	notification1, ok := <-notificationC1
	jobStatus := JobStatus{Name: DOWNLOADING, Err: nil, Offset: 4}
	AssertTrue(reflect.DeepEqual(jobStatus, notification1))
	AssertEq(true, ok)
	// check 2nd subscriber's offset
	AssertEq(subscriberOffset2, jt.job.subscribers.Front().Value.(jobSubscriber).subscribedOffset)
}

func (jt *jobTest) Test_failWhileDownloading() {
	subscriberOffset1 := int64(3)
	subscriberOffset2 := int64(5)
	notificationC1 := jt.job.subscribe(subscriberOffset1)
	notificationC2 := jt.job.subscribe(subscriberOffset2)
	jt.job.status = JobStatus{Name: DOWNLOADING, Err: nil, Offset: 4}

	customErr := fmt.Errorf("custom error")
	jt.job.failWhileDownloading(customErr)

	AssertEq(0, jt.job.subscribers.Len())
	notification1, ok1 := <-notificationC1
	notification2, ok2 := <-notificationC2
	jobStatus := JobStatus{Name: FAILED, Err: customErr, Offset: 4}
	// Check 1st and 2nd subscriber notifications
	AssertTrue(reflect.DeepEqual(jobStatus, notification1))
	AssertEq(true, ok1)
	AssertTrue(reflect.DeepEqual(jobStatus, notification2))
	AssertEq(true, ok2)
}

func (jt *jobTest) Test_updateFileInfoCache_UpdateEntry() {
	// Add an entry into
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: DefaultObjectName,
	}
	fileInfo := data.FileInfo{
		Key:              fileInfoKey,
		ObjectGeneration: jt.job.object.Generation,
		FileSize:         jt.job.object.Size,
		Offset:           0,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	_, err = jt.cache.Insert(fileInfoKeyName, fileInfo)
	AssertEq(nil, err)
	jt.job.status.Offset = 1

	err = jt.job.updateFileInfoCache()

	AssertEq(nil, err)
	// confirm fileInfoCache is updated with new offset.
	lookupResult := jt.cache.LookUp(fileInfoKeyName)
	AssertFalse(lookupResult == nil)
	fileInfo = lookupResult.(data.FileInfo)
	AssertEq(1, fileInfo.Offset)
	AssertEq(jt.job.object.Generation, fileInfo.ObjectGeneration)
	AssertEq(jt.job.object.Size, fileInfo.FileSize)
}

// This test should fail when we shift to only updating fileInfoCache in Job.
// This test should be removed when that happens.
func (jt *jobTest) Test_updateFileInfoCache_InsertNew() {
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: DefaultObjectName,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	jt.job.status.Offset = 1

	err = jt.job.updateFileInfoCache()

	AssertEq(nil, err)
	// confirm fileInfoCache is updated with new offset.
	lookupResult := jt.cache.LookUp(fileInfoKeyName)
	AssertFalse(lookupResult == nil)
	fileInfo := lookupResult.(data.FileInfo)
	AssertEq(1, fileInfo.Offset)
	AssertEq(jt.job.object.Generation, fileInfo.ObjectGeneration)
	AssertEq(jt.job.object.Size, fileInfo.FileSize)
}

func (jt *jobTest) Test_updateFileInfoCache_Fail() {
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: DefaultObjectName,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	AssertEq(nil, err)
	// set size of object more than MaxSize of cache.
	jt.job.object.Size = 100

	err = jt.job.updateFileInfoCache()

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), lru.InvalidEntrySizeErrorMsg))
	// confirm fileInfoCache is not updated.
	lookupResult := jt.cache.LookUp(fileInfoKeyName)
	AssertTrue(lookupResult == nil)
}

// Note: We can't test Test_downloadObjectAsync_MoreThanSequentialReadSize as
// the fake storage bucket/server in the testing environment doesn't support
// reading ranges (start and limit in NewReader call)
func (jt *jobTest) Test_downloadObjectAsync_LessThanSequentialReadSize() {
	// Create new object in bucket and create new job for it.
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize))

	// start download
	jt.job.downloadObjectAsync()

	// check job completed successfully
	jobStatus := JobStatus{COMPLETED, nil, int64(objectSize)}
	jt.job.mu.Lock()
	defer jt.job.mu.Unlock()
	AssertTrue(reflect.DeepEqual(jobStatus, jt.job.status))
	// verify file is downloaded
	jt.verifyFile(objectContent)
	// Verify fileInfoCache update
	jt.verifyFileInfoEntry(uint64(objectSize))
}

func (jt *jobTest) Test_downloadObjectAsync_LessThanChunkSize() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 2 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, 25, uint64(2*objectSize))

	// start download
	jt.job.downloadObjectAsync()

	// check job completed successfully
	jobStatus := JobStatus{COMPLETED, nil, int64(objectSize)}
	jt.job.mu.Lock()
	defer jt.job.mu.Unlock()
	AssertTrue(reflect.DeepEqual(jobStatus, jt.job.status))
	// verify file is downloaded
	jt.verifyFile(objectContent)
	// Verify fileInfoCache update
	jt.verifyFileInfoEntry(uint64(objectSize))
}

func (jt *jobTest) Test_downloadObjectAsync_Notification() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize))
	// Add subscriber
	subscribedOffset := int64(45 * util.MiB)
	notificationC := jt.job.subscribe(subscribedOffset)

	// start download
	jt.job.downloadObjectAsync()

	jobStatus := <-notificationC
	// check the notification is sent after subscribed offset
	AssertGe(jobStatus.Offset, subscribedOffset)
	// check job completed successfully
	jobStatus = JobStatus{COMPLETED, nil, int64(objectSize)}
	jt.job.mu.Lock()
	defer jt.job.mu.Unlock()
	AssertTrue(reflect.DeepEqual(jobStatus, jt.job.status))
	// verify file is downloaded
	jt.verifyFile(objectContent)
	// Verify fileInfoCache update
	jt.verifyFileInfoEntry(uint64(objectSize))
}

func (jt *jobTest) Test_downloadObjectAsync_ErrorWhenFileCacheHasLessSize() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize-1))

	// start download
	jt.job.downloadObjectAsync()

	// check job failed
	jt.job.mu.Lock()
	defer jt.job.mu.Unlock()
	AssertEq(FAILED, jt.job.status.Name)
	AssertTrue(strings.Contains(jt.job.status.Err.Error(), "size of the entry is more than the cache's maxSize"))
}

func (jt *jobTest) Test_Download_WhenNotStarted() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize))

	// start download
	offset := int64(25 * util.MiB)
	jobStatus, err := jt.job.Download(context.Background(), offset, true)

	AssertEq(nil, err)
	// verify that jobStatus is downloading and downloaded more than 25 Mib.
	AssertEq(DOWNLOADING, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, offset)
	// verify that after some time the whole object is downloaded
	time.Sleep(time.Second * 2)
	// verify file
	jt.verifyFile(objectContent)
	// verify file info cache
	jt.verifyFileInfoEntry(uint64(objectSize))
}

func (jt *jobTest) Test_Download_WhenAlreadyDownloading() {
	// Create new object in bucket and create new job for it.
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize))
	// start download but not wait for download
	ctx := context.Background()
	jobStatus, err := jt.job.Download(ctx, 1, false)
	AssertEq(nil, err)
	AssertEq(DOWNLOADING, jobStatus.Name)

	// Again call download but wait for download this time.
	offset := int64(25 * util.MiB)
	jobStatus, err = jt.job.Download(ctx, offset, true)

	AssertEq(nil, err)
	// verify that jobStatus is downloading and downloaded at least 25 Mib.
	AssertEq(DOWNLOADING, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, offset)
	// verify that after some time the whole object is downloaded
	time.Sleep(time.Second * 2)
	// verify file
	jt.verifyFile(objectContent)
	// verify file info cache
	jt.verifyFileInfoEntry(uint64(objectSize))
}

func (jt *jobTest) Test_Download_WhenAlreadyCompleted() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(2*objectSize))
	// Wait for whole download to be completed.
	ctx := context.Background()
	jobStatus, err := jt.job.Download(ctx, int64(objectSize), true)
	AssertEq(nil, err)
	// verify that jobStatus is DOWNLOADING but offset is object size
	expectedJobStatus := JobStatus{DOWNLOADING, nil, int64(objectSize)}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))

	// try to request for some offset when job was already completed.
	offset := int64(18 * util.MiB)
	jobStatus, err = jt.job.Download(ctx, offset, true)

	AssertEq(nil, err)
	// verify that jobStatus is completed & offset returned is still 25 MiB
	// this ensures that async job is not started again.
	AssertEq(COMPLETED, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, objectSize)
	// verify file
	jt.verifyFile(objectContent)
	// verify file info cache
	jt.verifyFileInfoEntry(uint64(jobStatus.Offset))
}

func (jt *jobTest) Test_Download_WhenAsyncFails() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	// set size of cache smaller than object size.
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize-1))

	// Wait for whole download to be completed/failed.
	ctx := context.Background()
	jobStatus, err := jt.job.Download(ctx, int64(objectSize), true)

	AssertEq(nil, err)
	// verify that jobStatus is failed
	AssertEq(FAILED, jobStatus.Name)
	AssertEq(ReadChunkSize, jobStatus.Offset)
	AssertTrue(strings.Contains(jobStatus.Err.Error(), "size of the entry is more than the cache's maxSize"))
}

func (jt *jobTest) Test_Download_AlreadyFailed() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize-1))
	// Wait for whole download to be completed/failed.
	jobStatus, err := jt.job.Download(context.Background(), int64(objectSize), true)
	AssertEq(nil, err)
	// verify that jobStatus is failed
	AssertEq(FAILED, jobStatus.Name)
	AssertEq(ReadChunkSize, jobStatus.Offset)
	AssertTrue(strings.Contains(jobStatus.Err.Error(), "size of the entry is more than the cache's maxSize"))

	// requesting again from download job which is in failed state
	jobStatus, err = jt.job.Download(context.Background(), int64(objectSize), true)

	AssertEq(nil, err)
	AssertEq(FAILED, jobStatus.Name)
	AssertEq(ReadChunkSize, jobStatus.Offset)
	AssertTrue(strings.Contains(jobStatus.Err.Error(), "size of the entry is more than the cache's maxSize"))
}

func (jt *jobTest) Test_Download_AlreadyInvalid() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 1 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize-1))
	// make the state as invalid
	jt.job.mu.Lock()
	jt.job.status.Name = INVALID
	jt.job.mu.Unlock()

	// requesting download when already invalid.
	jobStatus, err := jt.job.Download(context.Background(), int64(objectSize), true)

	AssertEq(nil, err)
	AssertEq(INVALID, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
}

func (jt *jobTest) Test_Download_InvalidOffset() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize))

	// requesting invalid offset
	offset := int64(objectSize) + 1
	jobStatus, err := jt.job.Download(context.Background(), offset, true)

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), fmt.Sprintf("Download: the requested offset %d is greater than the size of object %d", offset, jt.object.Size)))
	expectedJobStatus := JobStatus{NOT_STARTED, nil, 0}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))
}

func (jt *jobTest) Test_Download_CtxCancelled() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))

	// requesting full download and then the download call should be cancelled after
	// timeout but async download shouldn't be cancelled
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Millisecond*2)
	defer cancelFunc()
	offset := int64(objectSize)
	jobStatus, err := jt.job.Download(ctx, offset, true)

	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "context deadline exceeded"))
	// jobStatus is empty in this case.
	AssertEq("", jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	// job should be completed after sometime as the timeout is on Download call
	// and not async download
	time.Sleep(time.Second * 2)
	jobStatus, err = jt.job.Download(context.Background(), 0, false)
	AssertEq(nil, err)
	expectedJobStatus := JobStatus{COMPLETED, nil, int64(objectSize)}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))
	// verify file
	jt.verifyFile(objectContent)
	// verify file info cache
	jt.verifyFileInfoEntry(uint64(objectSize))
}

func (jt *jobTest) Test_Download_Concurrent() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	ctx := context.Background()
	wg := sync.WaitGroup{}
	offsets := []int64{0, 4 * util.MiB, 16 * util.MiB, 8 * util.MiB, int64(objectSize), int64(objectSize) + 1}
	expectedErrs := []error{nil, nil, nil, nil, nil, fmt.Errorf(fmt.Sprintf("Download: the requested offset %d is greater than the size of object %d", int64(objectSize)+1, int64(objectSize)))}
	downloadFunc := func(expectedOffset int64, expectedErr error) {
		defer wg.Done()
		var jobStatus JobStatus
		var err error
		jobStatus, err = jt.job.Download(ctx, expectedOffset, true)
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

	jobStatus, err := jt.job.Download(context.Background(), 0, false)
	AssertEq(nil, err)
	expectedJobStatus := JobStatus{COMPLETED, nil, int64(objectSize)}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))
	// verify file
	jt.verifyFile(objectContent)
	// verify file info cache
	jt.verifyFileInfoEntry(uint64(objectSize))
}

func (jt *jobTest) Test_Cancel_WhenDownlooading() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	// request for 2 MiB download to start downloading
	offset := int64(2 * util.MiB)
	jobStatus, err := jt.job.Download(context.Background(), offset, true)
	AssertEq(nil, err)
	AssertEq(DOWNLOADING, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, offset)

	// Wait for some time and then cancel
	time.Sleep(time.Millisecond * 30)
	jt.job.Cancel()

	jobStatus, err = jt.job.Download(context.Background(), 0, false)
	AssertEq(nil, err)
	AssertEq(CANCELLED, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	// file is not completely downloaded as job was still running when cancelled.
	AssertLt(jobStatus.Offset, objectSize)
	// job should not be completed even after sometime.
	time.Sleep(time.Second)
	newJobStatus, err := jt.job.Download(context.Background(), 0, false)
	AssertEq(nil, err)
	AssertEq(CANCELLED, newJobStatus.Name)
	AssertEq(nil, newJobStatus.Err)
	// file is not completely downloaded as job was still running when cancelled.
	AssertLt(newJobStatus.Offset, objectSize)
	// verify file downloaded till the offset
	jt.verifyFile(objectContent[:newJobStatus.Offset])
}

func (jt *jobTest) Test_Cancel_WhenAlreadyCompleted() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	// download complete object
	jobStatus, err := jt.job.Download(context.Background(), int64(objectSize), true)
	AssertEq(nil, err)
	expectedJobStatus := JobStatus{DOWNLOADING, nil, int64(objectSize)}
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))

	jt.job.Cancel()

	// status is not changed to Cancelled
	jobStatus, err = jt.job.Download(context.Background(), 0, false)
	AssertEq(nil, err)
	AssertNe(CANCELLED, jobStatus.Name)
	AssertEq(nil, err)
	AssertEq(objectSize, jobStatus.Offset)
	// verify file downloaded till the offset
	jt.verifyFile(objectContent)
	// verify file info cache
	jt.verifyFileInfoEntry(uint64(objectSize))
}

func (jt *jobTest) Test_Cancel_WhenNotStarted() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))

	jt.job.Cancel()

	// status is changed to Cancelled
	expectedJobStatus := JobStatus{CANCELLED, nil, 0}
	jobStatus, err := jt.job.Download(context.Background(), 0, false)
	AssertEq(nil, err)
	AssertTrue(reflect.DeepEqual(expectedJobStatus, jobStatus))
	// verify file is not created
	_, err = os.Stat(jt.fileSpec.Path)
	AssertNe(nil, err)
	AssertTrue(strings.Contains(err.Error(), "no such file or directory"))
}

func (jt *jobTest) Test_Cancel_Concurrent() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	ctx := context.Background()
	// start download without waiting
	jobStatus, err := jt.job.Download(ctx, 0, false)
	AssertEq(nil, err)
	AssertEq(DOWNLOADING, jobStatus.Name)
	// wait for sometime to allow downloading before cancelling
	time.Sleep(time.Millisecond * 10)
	wg := sync.WaitGroup{}
	cancelFunc := func() {
		defer wg.Done()
		jt.job.Cancel()
		currJobStatus, currErr := jt.job.Download(ctx, 1, true)
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

	jobStatus, err = jt.job.Download(ctx, 1, true)
	AssertEq(nil, err)
	AssertEq(CANCELLED, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	// verify file
	jt.verifyFile(objectContent[:jobStatus.Offset])
}

func (jt *jobTest) Test_GetStatus() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 20 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	ctx := context.Background()
	// start download without waiting
	jobStatus, err := jt.job.Download(ctx, 0, false)
	AssertEq(nil, err)
	AssertEq(DOWNLOADING, jobStatus.Name)
	// wait for sometime to get some data downloaded
	time.Sleep(10 * time.Millisecond)

	// GetStatus in between downloading
	jobStatus = jt.job.GetStatus()

	AssertEq(DOWNLOADING, jobStatus.Name)
	AssertEq(nil, jobStatus.Err)
	AssertGe(jobStatus.Offset, 0)
	// verify file
	jt.verifyFile(objectContent[:jobStatus.Offset])
}

func (jt *jobTest) Test_Invalidate_WhenDownloading() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 25 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	ctx := context.Background()
	// start download without waiting
	jobStatus, err := jt.job.Download(ctx, 0, false)
	AssertEq(nil, err)
	AssertEq(DOWNLOADING, jobStatus.Name)

	jt.job.Invalidate()

	jobStatus = jt.job.GetStatus()
	AssertEq(nil, jobStatus.Err)
	AssertEq(INVALID, jobStatus.Name)
}

func (jt *jobTest) Test_Invalidate_Concurrent() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 30 * util.MiB
	objectContent := generateRandomBytes(objectSize)
	jt.initJobTest(objectName, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize*2))
	ctx := context.Background()
	// start download without waiting
	jobStatus, err := jt.job.Download(ctx, 0, false)
	AssertEq(nil, err)
	AssertEq(DOWNLOADING, jobStatus.Name)
	wg := sync.WaitGroup{}
	invalidateFunc := func() {
		defer wg.Done()
		jt.job.Invalidate()
		currJobStatus := jt.job.GetStatus()
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
