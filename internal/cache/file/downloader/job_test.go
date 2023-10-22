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
	"bytes"
	"container/list"
	"context"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
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
const DefaultSequentialReadSizeMb = 200

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
	ExpectEq(nil, err)
	jt.object = jt.getMinObject(objectName)
	jt.fileSpec = data.FileSpec{Path: jt.fileCachePath(jt.bucket.Name(), jt.object.Name), Perm: os.FileMode(0644)}
	jt.cache = lru.NewCache(lruCacheSize)
	jt.job = NewJob(&jt.object, jt.bucket, jt.cache, sequentialReadSize, jt.fileSpec)
}

func (jt *jobTest) verifyFile(content []byte) {
	fileStat, err := os.Stat(jt.fileSpec.Path)
	ExpectEq(nil, err)
	ExpectEq(jt.fileSpec.Perm, fileStat.Mode())
	ExpectEq(len(content), fileStat.Size())
	// verify the content of file downloaded. only verified till
	fileContent, err := os.ReadFile(jt.fileSpec.Path)
	ExpectEq(nil, err)
	ExpectTrue(reflect.DeepEqual(content, fileContent[:len(content)]))
}

func (jt *jobTest) verifyFileInfoEntry(offset uint64) {
	fileInfoKey := data.FileInfoKey{BucketName: jt.bucket.Name(), ObjectName: jt.object.Name}
	fileInfoKeyName, err := fileInfoKey.Key()
	ExpectEq(nil, err)
	fileInfo := jt.cache.LookUp(fileInfoKeyName)
	ExpectTrue(fileInfo != nil)
	ExpectEq(jt.object.Generation, fileInfo.(data.FileInfo).ObjectGeneration)
	ExpectEq(offset, fileInfo.(data.FileInfo).Offset)
	ExpectEq(jt.object.Size, fileInfo.(data.FileInfo).Size())
}

func (jt *jobTest) fileCachePath(bucketName string, objectName string) string {
	return path.Join(cacheLocation, bucketName, objectName)
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

	ExpectEq(NOT_STARTED, jt.job.status.Name)
	ExpectEq(nil, jt.job.status.Err)
	ExpectEq(0, jt.job.status.Offset)
	ExpectTrue(reflect.DeepEqual(list.List{}, jt.job.subscribers))
	ExpectNe(nil, jt.job.cancelCtx)
	ExpectNe(nil, jt.job.cancelFunc)
}

func (jt *jobTest) Test_subscribe() {
	subscriberOffset1 := int64(0)
	subscriberOffset2 := int64(1)

	notificationC1 := jt.job.subscribe(subscriberOffset1)
	notificationC2 := jt.job.subscribe(subscriberOffset2)

	ExpectEq(2, jt.job.subscribers.Len())
	receivingC := make(<-chan JobStatus, 1)
	ExpectEq(reflect.TypeOf(receivingC), reflect.TypeOf(notificationC1))
	ExpectEq(reflect.TypeOf(receivingC), reflect.TypeOf(notificationC2))
	// check 1st and 2nd subscribers
	var subscriber jobSubscriber
	ExpectEq(reflect.TypeOf(subscriber), reflect.TypeOf(jt.job.subscribers.Front().Value.(jobSubscriber)))
	ExpectEq(0, jt.job.subscribers.Front().Value.(jobSubscriber).subscribedOffset)
	ExpectEq(reflect.TypeOf(subscriber), reflect.TypeOf(jt.job.subscribers.Back().Value.(jobSubscriber)))
	ExpectEq(1, jt.job.subscribers.Back().Value.(jobSubscriber).subscribedOffset)
}

func (jt *jobTest) Test_notifySubscriber_FAILED() {
	subscriberOffset := int64(1)
	notificationC := jt.job.subscribe(subscriberOffset)
	jt.job.status.Name = FAILED
	customErr := fmt.Errorf("custom err")
	jt.job.status.Err = customErr

	jt.job.notifySubscribers()

	ExpectEq(0, jt.job.subscribers.Len())
	notification, ok := <-notificationC
	jobStatus := JobStatus{Name: FAILED, Err: customErr, Offset: 0}
	ExpectTrue(reflect.DeepEqual(jobStatus, notification))
	ExpectEq(true, ok)
}

func (jt *jobTest) Test_notifySubscriber_CANCELLED() {
	subscriberOffset := int64(1)
	notificationC := jt.job.subscribe(subscriberOffset)
	jt.job.status.Name = CANCELLED

	jt.job.notifySubscribers()

	ExpectEq(0, jt.job.subscribers.Len())
	notification, ok := <-notificationC
	jobStatus := JobStatus{Name: CANCELLED, Err: nil, Offset: 0}
	ExpectTrue(reflect.DeepEqual(jobStatus, notification))
	ExpectEq(true, ok)
}

func (jt *jobTest) Test_notifySubscriber_SubscribedOffset() {
	subscriberOffset1 := int64(3)
	subscriberOffset2 := int64(5)
	notificationC1 := jt.job.subscribe(subscriberOffset1)
	_ = jt.job.subscribe(subscriberOffset2)
	jt.job.status.Name = DOWNLOADING
	jt.job.status.Offset = 4

	jt.job.notifySubscribers()

	ExpectEq(1, jt.job.subscribers.Len())
	notification1, ok := <-notificationC1
	jobStatus := JobStatus{Name: DOWNLOADING, Err: nil, Offset: 4}
	ExpectTrue(reflect.DeepEqual(jobStatus, notification1))
	ExpectEq(true, ok)
	// check 2nd subscriber's offset
	ExpectEq(subscriberOffset2, jt.job.subscribers.Front().Value.(jobSubscriber).subscribedOffset)
}

func (jt *jobTest) Test_failWhileDownloading() {
	subscriberOffset1 := int64(3)
	subscriberOffset2 := int64(5)
	notificationC1 := jt.job.subscribe(subscriberOffset1)
	notificationC2 := jt.job.subscribe(subscriberOffset2)
	jt.job.status = JobStatus{Name: DOWNLOADING, Err: nil, Offset: 4}

	customErr := fmt.Errorf("custom error")
	jt.job.failWhileDownloading(customErr)

	ExpectEq(0, jt.job.subscribers.Len())
	notification1, ok1 := <-notificationC1
	notification2, ok2 := <-notificationC2
	jobStatus := JobStatus{Name: FAILED, Err: customErr, Offset: 4}
	// Check 1st and 2nd subscriber notifications
	ExpectTrue(reflect.DeepEqual(jobStatus, notification1))
	ExpectEq(true, ok1)
	ExpectTrue(reflect.DeepEqual(jobStatus, notification2))
	ExpectEq(true, ok2)
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
	ExpectEq(nil, err)
	_, err = jt.cache.Insert(fileInfoKeyName, fileInfo)
	ExpectEq(nil, err)
	jt.job.status.Offset = 1

	err = jt.job.updateFileInfoCache()

	ExpectEq(nil, err)
	// confirm fileInfoCache is updated with new offset.
	lookupResult := jt.cache.LookUp(fileInfoKeyName)
	ExpectFalse(lookupResult == nil)
	fileInfo = lookupResult.(data.FileInfo)
	ExpectEq(1, fileInfo.Offset)
	ExpectEq(jt.job.object.Generation, fileInfo.ObjectGeneration)
	ExpectEq(jt.job.object.Size, fileInfo.FileSize)
}

// This test should fail when we shift to only updating fileInfoCache in Job.
// This test should be removed when that happens.
func (jt *jobTest) Test_updateFileInfoCache_InsertNew() {
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: DefaultObjectName,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	ExpectEq(nil, err)
	jt.job.status.Offset = 1

	err = jt.job.updateFileInfoCache()

	ExpectEq(nil, err)
	// confirm fileInfoCache is updated with new offset.
	lookupResult := jt.cache.LookUp(fileInfoKeyName)
	ExpectFalse(lookupResult == nil)
	fileInfo := lookupResult.(data.FileInfo)
	ExpectEq(1, fileInfo.Offset)
	ExpectEq(jt.job.object.Generation, fileInfo.ObjectGeneration)
	ExpectEq(jt.job.object.Size, fileInfo.FileSize)
}

func (jt *jobTest) Test_updateFileInfoCache_Fail() {
	fileInfoKey := data.FileInfoKey{
		BucketName: storage.TestBucketName,
		ObjectName: DefaultObjectName,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	ExpectEq(nil, err)
	// set size of object more than MaxSize of cache.
	jt.job.object.Size = 100

	err = jt.job.updateFileInfoCache()

	ExpectNe(nil, err)
	ExpectTrue(strings.Contains(err.Error(), lru.InvalidEntrySizeErrorMsg))
	// confirm fileInfoCache is not updated.
	lookupResult := jt.cache.LookUp(fileInfoKeyName)
	ExpectTrue(lookupResult == nil)
}

// Note: We can't test Test_downloadObjectAsync_MoreThanSequentialReadSize as
// the fake storage bucket/server in the testing environment doesn't support
// reading ranges (start and limit in NewReader call)
func (jt *jobTest) Test_downloadObjectAsync_LessThanSequentialReadSize() {
	// Create new object in bucket and create new job for it.
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * MiB
	objectContent := bytes.Repeat([]byte("t"), objectSize)
	jt.initJobTest(objectName, objectContent, 100, uint64(2*objectSize))

	// start download
	jt.job.downloadObjectAsync()

	// check job completed successfully
	jobStatus := JobStatus{COMPLETED, nil, int64(objectSize)}
	jt.job.mu.Lock()
	defer jt.job.mu.Unlock()
	ExpectTrue(reflect.DeepEqual(jobStatus, jt.job.status))
	// verify file is downloaded
	jt.verifyFile(objectContent)
	// Verify fileInfoCache update
	jt.verifyFileInfoEntry(uint64(objectSize))
}

func (jt *jobTest) Test_downloadObjectAsync_LessThanChunkSize() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 2 * MiB
	objectContent := bytes.Repeat([]byte("t"), objectSize)
	jt.initJobTest(objectName, objectContent, 25, uint64(2*objectSize))

	// start download
	jt.job.downloadObjectAsync()

	// check job completed successfully
	jobStatus := JobStatus{COMPLETED, nil, int64(objectSize)}
	jt.job.mu.Lock()
	defer jt.job.mu.Unlock()
	ExpectTrue(reflect.DeepEqual(jobStatus, jt.job.status))
	// verify file is downloaded
	jt.verifyFile(objectContent)
	// Verify fileInfoCache update
	jt.verifyFileInfoEntry(uint64(objectSize))
}

func (jt *jobTest) Test_downloadObjectAsync_Notification() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * MiB
	objectContent := bytes.Repeat([]byte("t"), objectSize)
	jt.initJobTest(objectName, objectContent, 100, uint64(2*objectSize))
	// Add subscriber
	subscribedOffset := int64(45 * MiB)
	notificationC := jt.job.subscribe(subscribedOffset)

	// start download
	jt.job.downloadObjectAsync()

	jobStatus := <-notificationC
	// check the notification is sent after subscribed offset
	ExpectGe(jobStatus.Offset, subscribedOffset)
	// check job completed successfully
	jobStatus = JobStatus{COMPLETED, nil, int64(objectSize)}
	jt.job.mu.Lock()
	defer jt.job.mu.Unlock()
	ExpectTrue(reflect.DeepEqual(jobStatus, jt.job.status))
	// verify file is downloaded
	jt.verifyFile(objectContent)
	// Verify fileInfoCache update
	jt.verifyFileInfoEntry(uint64(objectSize))
}

func (jt *jobTest) Test_downloadObjectAsync_ErrorWhenFileCacheHasLessSize() {
	objectName := "path/in/gcs/foo.txt"
	objectSize := 50 * MiB
	objectContent := bytes.Repeat([]byte("t"), objectSize)
	jt.initJobTest(objectName, objectContent, 100, uint64(objectSize-1))

	// start download
	jt.job.downloadObjectAsync()

	// check job failed
	jt.job.mu.Lock()
	defer jt.job.mu.Unlock()
	ExpectEq(FAILED, jt.job.status.Name)
	ExpectEq(ReadChunkSize, jt.job.status.Offset)
	ExpectTrue(strings.Contains(jt.job.status.Err.Error(), "size of the entry is more than the cache's maxSize"))
}
