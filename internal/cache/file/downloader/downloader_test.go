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
	"os"
	"path"
	"reflect"
	"sync"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	. "github.com/jacobsa/ogletest"
	"golang.org/x/net/context"
)

var cacheLocation string = path.Join(os.Getenv("HOME"), "cache/location")

func TestDownloader(t *testing.T) { RunTests(t) }

type downloaderTest struct {
	job         *Job
	bucket      gcs.Bucket
	object      gcs.MinObject
	cache       *lru.Cache
	fakeStorage storage.FakeStorage
	fileSpec    data.FileSpec
	jm          *JobManager
}

func init() { RegisterTestSuite(&downloaderTest{}) }

func (dt *downloaderTest) SetUp(*TestInfo) {
	locker.EnableInvariantsCheck()
	operations.RemoveDir(cacheLocation)

	// Create bucket in fake storage.
	dt.fakeStorage = storage.NewFakeStorage()
	storageHandle := dt.fakeStorage.CreateStorageHandle()
	dt.bucket = storageHandle.BucketHandle(storage.TestBucketName, "")

	dt.initJobTest(DefaultObjectName, []byte("taco"), 200, CacheMaxSize)
	dt.jm = NewJobManager(dt.cache, util.DefaultFilePerm, cacheLocation, DefaultSequentialReadSizeMb)
}

func (dt *downloaderTest) TearDown() {
	dt.fakeStorage.ShutDown()
	operations.RemoveDir(cacheLocation)
}

func (dt *downloaderTest) verifyJob(job *Job, object *gcs.MinObject, bucket gcs.Bucket, sequentialReadSizeMb int32) {
	job.mu.Lock()
	defer job.mu.Unlock()
	ExpectEq(object.Generation, job.object.Generation)
	ExpectEq(object.Name, job.object.Name)
	ExpectEq(bucket.Name(), job.bucket.Name())
	downloadPath := dt.jm.getDownloadPath(getObjectPath(bucket.Name(), object.Name))
	ExpectEq(downloadPath, job.fileSpec.Path)
	ExpectEq(sequentialReadSizeMb, job.sequentialReadSizeMb)
}

func (dt *downloaderTest) Test_getObjectPath() {
	inputs := [][]string{{"", ""}, {"a", "b"}, {"a/b/", "/c/d"}, {"", "a"}, {"a", ""}}
	expectedOutPuts := [5]string{"", "a/b", "a/b/c/d", "a", "a"}

	results := [5]string{}
	for i := 0; i < 5; i++ {
		results[i] = getObjectPath(inputs[i][0], inputs[i][1])
	}

	ExpectTrue(reflect.DeepEqual(expectedOutPuts, results))
}

func (dt *downloaderTest) Test_getDownloadPath() {
	inputs := []string{"/", "a/b", "a/b/c/d", "/a", "a/"}
	dt.jm.mu.Lock()
	expectedOutputs := [5]string{dt.jm.cacheLocation, dt.jm.cacheLocation + "/a/b",
		dt.jm.cacheLocation + "/a/b/c/d", dt.jm.cacheLocation + "/a", dt.jm.cacheLocation + "/a"}
	dt.jm.mu.Unlock()

	results := [5]string{}
	for i := 0; i < 5; i++ {
		results[i] = dt.jm.getDownloadPath(inputs[i])
	}

	ExpectTrue(reflect.DeepEqual(expectedOutputs, results))
}

func (dt *downloaderTest) Test_GetJob_NotExisting() {

	// call GetJob for job which doesn't exist.
	job := dt.jm.GetJob(&dt.object, dt.bucket)

	dt.jm.mu.Lock()
	defer dt.jm.mu.Unlock()
	dt.verifyJob(job, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
}

func (dt *downloaderTest) Test_GetJob_Existing() {
	// first create new job
	expectedJob := dt.jm.GetJob(&dt.object, dt.bucket)
	dt.jm.mu.Lock()
	dt.verifyJob(expectedJob, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	dt.jm.mu.Unlock()

	// again call GetJob
	job := dt.jm.GetJob(&dt.object, dt.bucket)

	ExpectEq(expectedJob, job)
}

func (dt *downloaderTest) Test_GetJob_Concurrent() {
	jobs := [5]*Job{}
	wg := sync.WaitGroup{}
	getFunc := func(i int) {
		defer wg.Done()
		job := dt.jm.GetJob(&dt.object, dt.bucket)
		jobs[i] = job
	}

	// make concurrent requests
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go getFunc(i)
	}
	wg.Wait()

	// verify any of the job first
	dt.jm.mu.Lock()
	defer dt.jm.mu.Unlock()
	dt.verifyJob(jobs[0], &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	// verify all other jobs
	for i := 1; i < 5; i++ {
		ExpectEq(jobs[0], jobs[i])
	}
}

func (dt *downloaderTest) Test_RemoveJob_NotExisting() {
	// first create new job
	expectedJob := dt.jm.GetJob(&dt.object, dt.bucket)
	dt.jm.mu.Lock()
	dt.verifyJob(expectedJob, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	// verify the created
	dt.jm.mu.Unlock()

	dt.jm.RemoveJob(dt.object.Name, dt.bucket.Name())

	// verify no job existing
	dt.jm.mu.Lock()
	objectPath := getObjectPath(dt.bucket.Name(), dt.object.Name)
	_, ok := dt.jm.jobs[objectPath]
	ExpectFalse(ok)
	dt.jm.mu.Unlock()
}

func (dt *downloaderTest) Test_RemoveJob_Existing() {
	// first create new job
	expectedJob := dt.jm.GetJob(&dt.object, dt.bucket)
	dt.jm.mu.Lock()
	dt.verifyJob(expectedJob, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	dt.jm.mu.Unlock()
	// start the job
	_, _ = expectedJob.Download(context.Background(), 0, false)

	// remove the job
	dt.jm.RemoveJob(dt.object.Name, dt.bucket.Name())

	// verify no job existing
	dt.jm.mu.Lock()
	objectPath := getObjectPath(dt.bucket.Name(), dt.object.Name)
	_, ok := dt.jm.jobs[objectPath]
	ExpectFalse(ok)
	dt.jm.mu.Unlock()
	// verify the job is invalid
	jobStatus, err := expectedJob.Download(context.Background(), 0, false)
	ExpectEq(nil, err)
	ExpectEq(INVALID, jobStatus.Name)
}

func (dt *downloaderTest) Test_RemoveJob_Concurrent() {
	// first create new job
	expectedJob := dt.jm.GetJob(&dt.object, dt.bucket)
	dt.jm.mu.Lock()
	dt.verifyJob(expectedJob, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	dt.jm.mu.Unlock()
	wg := sync.WaitGroup{}

	// make concurrent requests
	for i := 0; i < 5; i++ {
		wg.Add(1)
		removeFunc := func() {
			wg.Done()
			dt.jm.RemoveJob(dt.object.Name, dt.bucket.Name())
		}
		go removeFunc()
	}
	wg.Wait()

	// verify no job existing
	dt.jm.mu.Lock()
	objectPath := getObjectPath(dt.bucket.Name(), dt.object.Name)
	_, ok := dt.jm.jobs[objectPath]
	ExpectFalse(ok)
	dt.jm.mu.Unlock()
}
