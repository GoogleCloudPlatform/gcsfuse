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
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/storage/control/apiv2/controlpb"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	testutil "github.com/googlecloudplatform/gcsfuse/v3/internal/util"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type downloaderTest struct {
	t                      *testing.T
	assert                 *assert.Assertions
	require                *require.Assertions
	defaultFileCacheConfig *cfg.FileCacheConfig
	job                    *Job
	bucket                 gcs.Bucket
	object                 gcs.MinObject
	cache                  *lru.Cache
	fakeStorage            storage.FakeStorage
	fileSpec               data.FileSpec
	jm                     *JobManager
	cacheDir               string
}

func newDownloaderTest(t *testing.T) *downloaderTest {
	dt := &downloaderTest{
		t:       t,
		assert:  assert.New(t),
		require: require.New(t),
	}
	dt.defaultFileCacheConfig = &cfg.FileCacheConfig{EnableCrc: true, ExperimentalParallelDownloadsDefaultOn: true}
	dt.setupHelper()
	return dt
}

func (dt *downloaderTest) setupHelper() {
	locker.EnableInvariantsCheck()
	dt.cacheDir = dt.t.TempDir()

	// Create bucket in fake storage.
	var err error
	mockClient := new(storage.MockStorageControlClient)
	dt.fakeStorage = storage.NewFakeStorageWithMockClient(mockClient, cfg.HTTP2)
	storageHandle := dt.fakeStorage.CreateStorageHandle()
	mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(&controlpb.StorageLayout{}, nil)
	ctx := context.Background()
	dt.bucket, err = storageHandle.BucketHandle(ctx, storage.TestBucketName, "")
	dt.require.NoError(err)

	dt.initJobTest(DefaultObjectName, []byte("taco"), DefaultSequentialReadSizeMb, CacheMaxSize, func() {})
	dt.jm = NewJobManager(dt.cache, util.DefaultFilePerm, util.DefaultDirPerm, dt.cacheDir, DefaultSequentialReadSizeMb, dt.defaultFileCacheConfig, metrics.NewNoopMetrics(), tracing.NewNoopTracer(), 1)
}

func (dt *downloaderTest) tearDown() {
	if dt.job != nil {
		dt.job.Invalidate()
	}
	if dt.jm != nil {
		dt.jm.Destroy()
	}
	dt.fakeStorage.ShutDown()
}

func (dt *downloaderTest) waitForCrcCheckToBeCompleted() {
	for {
		dt.job.mu.Lock()
		if dt.job.status.Name == Completed || dt.job.status.Name == Failed || dt.job.status.Name == Invalid {
			dt.job.mu.Unlock()
			break
		}
		dt.job.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
}

func (dt *downloaderTest) verifyJob(job *Job, object *gcs.MinObject, bucket gcs.Bucket, sequentialReadSizeMb int32) {
	dt.job.mu.Lock()
	defer dt.job.mu.Unlock()
	dt.assert.Equal(object.Generation, job.object.Generation)
	dt.assert.Equal(object.Name, job.object.Name)
	dt.assert.Equal(bucket.Name(), job.bucket.Name())
	downloadPath := util.GetDownloadPath(dt.jm.cacheDir, util.GetObjectPath(bucket.Name(), object.Name))
	dt.assert.Equal(downloadPath, job.fileSpec.Path)
	dt.assert.Equal(sequentialReadSizeMb, job.sequentialReadSizeMb)
	dt.assert.NotNil(job.removeJobCallback)
}

func Test_CreateJobIfNotExists_NotExisting(t *testing.T) {
	dt := newDownloaderTest(t)
	defer dt.tearDown()

	dt.jm.mu.Lock()
	objectPath := util.GetObjectPath(dt.bucket.Name(), dt.object.Name)
	_, ok := dt.jm.jobs[objectPath]
	dt.assert.False(ok)
	dt.jm.mu.Unlock()

	// Call CreateJobIfNotExists for job which doesn't exist.
	job := dt.jm.CreateJobIfNotExists(&dt.object, dt.bucket)

	dt.jm.mu.Lock()
	defer dt.jm.mu.Unlock()
	dt.verifyJob(job, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	actualJob, ok := dt.jm.jobs[objectPath]
	dt.assert.True(ok)
	dt.assert.Equal(job, actualJob)
}

func Test_CreateJobIfNotExists_Existing(t *testing.T) {
	dt := newDownloaderTest(t)
	defer dt.tearDown()

	// First create and store new job
	dt.jm.mu.Lock()
	objectPath := util.GetObjectPath(dt.bucket.Name(), dt.object.Name)
	dt.jm.jobs[objectPath] = dt.job
	dt.jm.mu.Unlock()

	// Call CreateJobIfNotExists for existing job.
	job := dt.jm.CreateJobIfNotExists(&dt.object, dt.bucket)

	dt.assert.Equal(dt.job, job)
	dt.jm.mu.Lock()
	defer dt.jm.mu.Unlock()
	dt.verifyJob(job, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	actualJob, ok := dt.jm.jobs[objectPath]
	dt.assert.True(ok)
	dt.assert.Equal(job, actualJob)
}

func Test_CreateJobIfNotExists_NotExisting_WithDefaultFileAndDirPerm(t *testing.T) {
	dt := newDownloaderTest(t)
	defer dt.tearDown()

	dt.jm.mu.Lock()
	objectPath := util.GetObjectPath(dt.bucket.Name(), dt.object.Name)
	_, ok := dt.jm.jobs[objectPath]
	dt.assert.False(ok)
	dt.jm.mu.Unlock()

	// Call CreateJobIfNotExists for job which doesn't exist.
	job := dt.jm.CreateJobIfNotExists(&dt.object, dt.bucket)

	dt.assert.Equal(os.FileMode(0700), job.fileSpec.DirPerm.Perm())
	dt.assert.Equal(os.FileMode(0600), job.fileSpec.FilePerm.Perm())
}

func Test_GetJob_NotExisting(t *testing.T) {
	dt := newDownloaderTest(t)
	defer dt.tearDown()

	dt.jm.mu.Lock()
	objectPath := util.GetObjectPath(dt.bucket.Name(), dt.object.Name)
	_, ok := dt.jm.jobs[objectPath]
	dt.assert.False(ok)
	dt.jm.mu.Unlock()

	// Call GetJob for job which doesn't exist.
	job := dt.jm.GetJob(dt.object.Name, dt.bucket.Name())

	dt.assert.Nil(job)
	dt.jm.mu.Lock()
	defer dt.jm.mu.Unlock()
	_, ok = dt.jm.jobs[objectPath]
	dt.assert.False(ok)
}

func Test_GetJob_Existing(t *testing.T) {
	dt := newDownloaderTest(t)
	defer dt.tearDown()

	// First create and store new job
	dt.jm.mu.Lock()
	objectPath := util.GetObjectPath(dt.bucket.Name(), dt.object.Name)
	dt.jm.jobs[objectPath] = dt.job
	dt.jm.mu.Unlock()

	// Call GetJob for existing job.
	job := dt.jm.GetJob(dt.object.Name, dt.bucket.Name())

	dt.assert.Equal(dt.job, job)
	dt.verifyJob(job, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
}

func Test_GetJob_Concurrent(t *testing.T) {
	dt := newDownloaderTest(t)
	defer dt.tearDown()

	// First create and store new job
	dt.jm.mu.Lock()
	objectPath := util.GetObjectPath(dt.bucket.Name(), dt.object.Name)
	dt.jm.jobs[objectPath] = dt.job
	dt.jm.mu.Unlock()
	jobs := [5]*Job{}
	wg := sync.WaitGroup{}
	getFunc := func(i int) {
		defer wg.Done()
		job := dt.jm.GetJob(dt.object.Name, dt.bucket.Name())
		jobs[i] = job
	}

	// make concurrent requests
	for i := range 5 {
		wg.Add(1)
		go getFunc(i)
	}
	wg.Wait()

	dt.verifyJob(dt.job, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	// Verify all jobs
	for i := range 5 {
		dt.assert.Equal(dt.job, jobs[i])
	}
}

func Test_InvalidateAndRemoveJob_NotExisting(t *testing.T) {
	dt := newDownloaderTest(t)
	defer dt.tearDown()

	expectedJob := dt.jm.GetJob(dt.object.Name, dt.bucket.Name())
	dt.assert.Nil(expectedJob)

	dt.jm.InvalidateAndRemoveJob(dt.object.Name, dt.bucket.Name())

	// Verify that job is invalidated and removed from job manager.
	expectedJob = dt.jm.GetJob(dt.object.Name, dt.bucket.Name())
	dt.assert.Nil(expectedJob)
}

func Test_InvalidateAndRemoveJob_Existing(t *testing.T) {
	dt := newDownloaderTest(t)
	defer dt.tearDown()

	// First create new job
	expectedJob := dt.jm.CreateJobIfNotExists(&dt.object, dt.bucket)
	dt.jm.mu.Lock()
	dt.verifyJob(expectedJob, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	dt.jm.mu.Unlock()
	// Start the job
	_, err := expectedJob.Download(context.Background(), 0, false)
	dt.require.NoError(err)

	// InvalidateAndRemove the job
	dt.jm.InvalidateAndRemoveJob(dt.object.Name, dt.bucket.Name())

	// Verify no job existing
	dt.assert.Equal(Invalid, expectedJob.GetStatus().Name)
	expectedJob = dt.jm.GetJob(dt.object.Name, dt.bucket.Name())
	dt.assert.Nil(expectedJob)
}

func Test_InvalidateAndRemoveJob_Concurrent(t *testing.T) {
	dt := newDownloaderTest(t)
	defer dt.tearDown()

	// First create new job
	expectedJob := dt.jm.CreateJobIfNotExists(&dt.object, dt.bucket)
	dt.jm.mu.Lock()
	dt.verifyJob(expectedJob, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	dt.jm.mu.Unlock()
	// Start the job
	_, err := expectedJob.Download(context.Background(), 0, false)
	dt.require.NoError(err)
	wg := sync.WaitGroup{}

	// Make concurrent requests
	for range 5 {
		wg.Add(1)
		invalidateFunc := func() {
			dt.jm.InvalidateAndRemoveJob(dt.object.Name, dt.bucket.Name())
			wg.Done()
		}
		go invalidateFunc()
	}
	wg.Wait()

	// Verify job in invalidated and removed from job manager.
	dt.assert.Equal(Invalid, expectedJob.GetStatus().Name)
	expectedJob = dt.jm.GetJob(dt.object.Name, dt.bucket.Name())
	dt.assert.Nil(expectedJob)
}

func Test_Destroy(t *testing.T) {
	dt := newDownloaderTest(t)
	defer dt.tearDown()

	objectSize := 50
	objectContent := testutil.GenerateRandomBytes(objectSize)
	// Create new jobs
	objectName1 := "path/in/gcs/foo1.txt"
	dt.initJobTest(objectName1, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize), func() {})
	object1 := dt.object
	job1 := dt.jm.CreateJobIfNotExists(&object1, dt.bucket)
	objectName2 := "path/in/gcs/foo2.txt"
	dt.initJobTest(objectName2, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize), func() {})
	object2 := dt.object
	job2 := dt.jm.CreateJobIfNotExists(&object2, dt.bucket)
	// Start the job
	_, err := job2.Download(context.Background(), 2, false)
	dt.require.NoError(err)
	objectName3 := "path/in/gcs/foo3.txt"
	dt.initJobTest(objectName3, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize), func() {})
	object3 := dt.object
	job3 := dt.jm.CreateJobIfNotExists(&object3, dt.bucket)
	// Start the job
	_, err = job3.Download(context.Background(), 2, false)
	dt.require.NoError(err)

	dt.jm.Destroy()

	// Verify all jobs are invalidated
	dt.assert.Equal(Invalid, job1.GetStatus().Name)
	dt.assert.Equal(Invalid, job2.GetStatus().Name)
	dt.assert.Equal(Invalid, job3.GetStatus().Name)
	// Verify all jobs are removed
	dt.assert.Nil(dt.jm.GetJob(objectName1, dt.bucket.Name()))
	dt.assert.Nil(dt.jm.GetJob(objectName2, dt.bucket.Name()))
	dt.assert.Nil(dt.jm.GetJob(objectName3, dt.bucket.Name()))
}

func Test_CreateJobIfNotExists_InvalidateAndRemoveJob_Concurrent(t *testing.T) {
	dt := newDownloaderTest(t)
	defer dt.tearDown()

	wg := sync.WaitGroup{}
	createNewJob := func() {
		job := dt.jm.CreateJobIfNotExists(&dt.object, dt.bucket)
		dt.assert.NotNil(job)
		wg.Done()
	}
	invalidateJob := func() {
		dt.jm.InvalidateAndRemoveJob(dt.object.Name, dt.bucket.Name())
		wg.Done()
	}

	for range 5 {
		wg.Add(2)
		go createNewJob()
		go invalidateJob()
	}
	wg.Wait()
}
