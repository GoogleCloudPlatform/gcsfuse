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
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
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

func TestJob(t *testing.T) { RunTests(t) }

type jobTest struct {
	job         *Job
	bucket      gcs.Bucket
	cache       *lru.Cache
	fakeStorage storage.FakeStorage
}

func init() { RegisterTestSuite(&jobTest{}) }

func (jt *jobTest) SetUp(*TestInfo) {
	locker.EnableInvariantsCheck()
	// Create bucket in fake storage.
	jt.fakeStorage = storage.NewFakeStorage()
	storageHandle := jt.fakeStorage.CreateStorageHandle()
	jt.bucket = storageHandle.BucketHandle(storage.TestBucketName, "")

	jt.cache = lru.NewCache(CacheMaxSize)

	ctx := context.Background()
	defaultObjects := map[string][]byte{
		// File
		DefaultObjectName: []byte("taco"),
		// Directory
		"bar/": []byte(""),
		// File
		"baz": []byte("burrito"),
	}
	err := storageutil.CreateObjects(ctx, jt.bucket, defaultObjects)
	if err != nil {
		panic(fmt.Errorf("error whlie creating objects: %v", err))
	}
	defaultObject, err := jt.bucket.StatObject(ctx, &gcs.StatObjectRequest{Name: DefaultObjectName,
		ForceFetchFromGcs: true})
	if err != nil {
		panic(fmt.Errorf("error whlie stating object: %v", err))
	}
	defaultMinObject := gcs.MinObject{
		Name:            defaultObject.Name,
		Size:            defaultObject.Size,
		Generation:      defaultObject.Generation,
		MetaGeneration:  defaultObject.MetaGeneration,
		Updated:         defaultObject.Updated,
		Metadata:        defaultObject.Metadata,
		ContentEncoding: defaultObject.ContentEncoding,
	}

	fileSpec := data.FileSpec{
		Path:     path.Join("./", storage.TestBucketName, DefaultObjectName),
		Perm:     os.FileMode(0666),
		OwnerUid: uint32(os.Getuid()),
		OwnerGid: uint32(os.Getgid()),
	}

	jt.job = NewJob(&defaultMinObject, jt.bucket, jt.cache, 200, fileSpec)
}

func (jt *jobTest) TearDown() {
	jt.fakeStorage.ShutDown()
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

func (jt *jobTest) Test_addSubscriber() {
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
