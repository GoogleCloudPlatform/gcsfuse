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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestDownloaderSuite(t *testing.T) {
	suite.Run(t, new(downloaderTest))
}

type downloaderTest struct {
	suite.Suite
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

func (dt *downloaderTest) setupHelper() {
	locker.EnableInvariantsCheck()
	// Create unique temp directory per test
	var err error
	dt.cacheDir, err = os.MkdirTemp("", "gcsfuse-test-*")
	require.NoError(dt.T(), err)

	// Create bucket in fake storage.
	mockClient := new(storage.MockStorageControlClient)
	dt.fakeStorage = storage.NewFakeStorageWithMockClient(mockClient, cfg.HTTP2)
	storageHandle := dt.fakeStorage.CreateStorageHandle()
	mockClient.On("GetStorageLayout", mock.Anything, mock.Anything, mock.Anything).
		Return(&controlpb.StorageLayout{}, nil)
	ctx := context.Background()
	dt.bucket, err = storageHandle.BucketHandle(ctx, storage.TestBucketName, "", false)
	require.NoError(dt.T(), err)

	dt.initJobTest(DefaultObjectName, []byte("taco"), DefaultSequentialReadSizeMb, CacheMaxSize, func() {})
	dt.jm = NewJobManager(dt.cache, util.DefaultFilePerm, util.DefaultDirPerm, dt.cacheDir, DefaultSequentialReadSizeMb, dt.defaultFileCacheConfig, metrics.NewNoopMetrics())
}

func (dt *downloaderTest) SetupTest() {
	dt.defaultFileCacheConfig = &cfg.FileCacheConfig{EnableCrc: true, ExperimentalParallelDownloadsDefaultOn: true}
	dt.setupHelper()
}

func (dt *downloaderTest) TearDownTest() {
	dt.fakeStorage.ShutDown()
	// Clean up temp dir
	if dt.cacheDir != "" {
		os.RemoveAll(dt.cacheDir)
	}
}

func (dt *downloaderTest) waitForCrcCheckToBeCompleted() {
	// Last notification is sent after the entire file is downloaded and before the CRC check is done.
	// Hence, explicitly waiting till the CRC check is done.
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			dt.job.mu.Lock()
			status := dt.job.status
			dt.job.mu.Unlock()
			require.Failf(dt.T(), "Timeout waiting for CRC check", "Current status: %v", status.Name)
			return
		case <-ticker.C:
			dt.job.mu.Lock()
			if dt.job.status.Name == Completed || dt.job.status.Name == Failed || dt.job.status.Name == Invalid {
				dt.job.mu.Unlock()
				return
			}
			dt.job.mu.Unlock()
		}
	}
}

func (dt *downloaderTest) verifyJob(job *Job, object *gcs.MinObject, bucket gcs.Bucket, sequentialReadSizeMb int32) {
	job.mu.Lock()
	defer job.mu.Unlock()
	assert.Equal(dt.T(), object.Generation, job.object.Generation)
	assert.Equal(dt.T(), object.Name, job.object.Name)
	assert.Equal(dt.T(), bucket.Name(), job.bucket.Name())
	downloadPath := util.GetDownloadPath(dt.jm.cacheDir, util.GetObjectPath(bucket.Name(), object.Name))
	assert.Equal(dt.T(), downloadPath, job.fileSpec.Path)
	assert.Equal(dt.T(), sequentialReadSizeMb, job.sequentialReadSizeMb)
	assert.NotNil(dt.T(), job.removeJobCallback)
}

func (dt *downloaderTest) Test_CreateJobIfNotExists_NotExisting() {
	dt.jm.mu.Lock()
	objectPath := util.GetObjectPath(dt.bucket.Name(), dt.object.Name)
	_, ok := dt.jm.jobs[objectPath]
	assert.False(dt.T(), ok)
	dt.jm.mu.Unlock()

	// Call CreateJobIfNotExists for job which doesn't exist.
	job := dt.jm.CreateJobIfNotExists(&dt.object, dt.bucket)

	dt.jm.mu.Lock()
	defer dt.jm.mu.Unlock()
	dt.verifyJob(job, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	actualJob, ok := dt.jm.jobs[objectPath]
	assert.True(dt.T(), ok)
	assert.Equal(dt.T(), job, actualJob)
}

func (dt *downloaderTest) Test_CreateJobIfNotExists_Existing() {
	// First create and store new job
	dt.jm.mu.Lock()
	objectPath := util.GetObjectPath(dt.bucket.Name(), dt.object.Name)
	dt.jm.jobs[objectPath] = dt.job
	dt.jm.mu.Unlock()

	// Call CreateJobIfNotExists for existing job.
	job := dt.jm.CreateJobIfNotExists(&dt.object, dt.bucket)

	assert.Equal(dt.T(), dt.job, job)
	dt.jm.mu.Lock()
	defer dt.jm.mu.Unlock()
	dt.verifyJob(job, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	actualJob, ok := dt.jm.jobs[objectPath]
	assert.True(dt.T(), ok)
	assert.Equal(dt.T(), job, actualJob)
}

func (dt *downloaderTest) Test_CreateJobIfNotExists_NotExisting_WithDefaultFileAndDirPerm() {
	dt.jm.mu.Lock()
	objectPath := util.GetObjectPath(dt.bucket.Name(), dt.object.Name)
	_, ok := dt.jm.jobs[objectPath]
	assert.False(dt.T(), ok)
	dt.jm.mu.Unlock()

	// Call CreateJobIfNotExists for job which doesn't exist.
	job := dt.jm.CreateJobIfNotExists(&dt.object, dt.bucket)

	assert.Equal(dt.T(), os.FileMode(0700), job.fileSpec.DirPerm.Perm())
	assert.Equal(dt.T(), os.FileMode(0600), job.fileSpec.FilePerm.Perm())
}

func (dt *downloaderTest) Test_GetJob_NotExisting() {
	dt.jm.mu.Lock()
	objectPath := util.GetObjectPath(dt.bucket.Name(), dt.object.Name)
	_, ok := dt.jm.jobs[objectPath]
	assert.False(dt.T(), ok)
	dt.jm.mu.Unlock()

	// Call GetJob for job which doesn't exist.
	job := dt.jm.GetJob(dt.object.Name, dt.bucket.Name())

	assert.Nil(dt.T(), job)
	dt.jm.mu.Lock()
	defer dt.jm.mu.Unlock()
	_, ok = dt.jm.jobs[objectPath]
	assert.False(dt.T(), ok)
}

func (dt *downloaderTest) Test_GetJob_Existing() {
	// First create and store new job
	dt.jm.mu.Lock()
	objectPath := util.GetObjectPath(dt.bucket.Name(), dt.object.Name)
	dt.jm.jobs[objectPath] = dt.job
	dt.jm.mu.Unlock()

	// Call GetJob for existing job.
	job := dt.jm.GetJob(dt.object.Name, dt.bucket.Name())

	assert.Equal(dt.T(), dt.job, job)
	dt.verifyJob(job, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
}

func (dt *downloaderTest) Test_GetJob_Concurrent() {
	// First create and store new job
	dt.jm.mu.Lock()
	objectPath := util.GetObjectPath(dt.bucket.Name(), dt.object.Name)
	dt.jm.jobs[objectPath] = dt.job
	dt.jm.mu.Unlock()
	jobs := make([]*Job, 5)
	var jobsMu sync.Mutex
	wg := sync.WaitGroup{}
	getFunc := func(i int) {
		defer wg.Done()
		job := dt.jm.GetJob(dt.object.Name, dt.bucket.Name())
		jobsMu.Lock()
		jobs[i] = job
		jobsMu.Unlock()
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
		assert.Equal(dt.T(), dt.job, jobs[i])
	}
}

func (dt *downloaderTest) Test_InvalidateAndRemoveJob_NotExisting() {
	expectedJob := dt.jm.GetJob(dt.object.Name, dt.bucket.Name())
	assert.Nil(dt.T(), expectedJob)

	dt.jm.InvalidateAndRemoveJob(dt.object.Name, dt.bucket.Name())

	// Verify that job is invalidated and removed from job manager.
	expectedJob = dt.jm.GetJob(dt.object.Name, dt.bucket.Name())
	assert.Nil(dt.T(), expectedJob)
}

func (dt *downloaderTest) Test_InvalidateAndRemoveJob_Existing() {
	// First create new job
	expectedJob := dt.jm.CreateJobIfNotExists(&dt.object, dt.bucket)
	dt.jm.mu.Lock()
	dt.verifyJob(expectedJob, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	dt.jm.mu.Unlock()
	// Start the job
	_, err := expectedJob.Download(context.Background(), 0, false)
	assert.Nil(dt.T(), err)

	// InvalidateAndRemove the job
	dt.jm.InvalidateAndRemoveJob(dt.object.Name, dt.bucket.Name())

	// Verify no job existing
	assert.Equal(dt.T(), Invalid, expectedJob.GetStatus().Name)
	expectedJob = dt.jm.GetJob(dt.object.Name, dt.bucket.Name())
	assert.Nil(dt.T(), expectedJob)
}

func (dt *downloaderTest) Test_InvalidateAndRemoveJob_Concurrent() {
	// First create new job
	expectedJob := dt.jm.CreateJobIfNotExists(&dt.object, dt.bucket)
	dt.jm.mu.Lock()
	dt.verifyJob(expectedJob, &dt.object, dt.bucket, dt.jm.sequentialReadSizeMb)
	dt.jm.mu.Unlock()
	// Start the job
	_, err := expectedJob.Download(context.Background(), 0, false)
	assert.Nil(dt.T(), err)
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
	assert.Equal(dt.T(), Invalid, expectedJob.GetStatus().Name)
	expectedJob = dt.jm.GetJob(dt.object.Name, dt.bucket.Name())
	assert.Nil(dt.T(), expectedJob)
}

func (dt *downloaderTest) Test_Destroy() {
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
	assert.Nil(dt.T(), err)
	objectName3 := "path/in/gcs/foo3.txt"
	dt.initJobTest(objectName3, objectContent, DefaultSequentialReadSizeMb, uint64(objectSize), func() {})
	object3 := dt.object
	job3 := dt.jm.CreateJobIfNotExists(&object3, dt.bucket)
	// Start the job
	_, err = job3.Download(context.Background(), 2, false)
	assert.Nil(dt.T(), err)

	dt.jm.Destroy()

	// Verify all jobs are invalidated
	assert.Equal(dt.T(), Invalid, job1.GetStatus().Name)
	assert.Equal(dt.T(), Invalid, job2.GetStatus().Name)
	assert.Equal(dt.T(), Invalid, job3.GetStatus().Name)
	// Verify all jobs are removed
	assert.Nil(dt.T(), dt.jm.GetJob(objectName1, dt.bucket.Name()))
	assert.Nil(dt.T(), dt.jm.GetJob(objectName2, dt.bucket.Name()))
	assert.Nil(dt.T(), dt.jm.GetJob(objectName3, dt.bucket.Name()))
}

func (dt *downloaderTest) Test_CreateJobIfNotExists_InvalidateAndRemoveJob_Concurrent() {
	wg := sync.WaitGroup{}
	createNewJob := func() {
		job := dt.jm.CreateJobIfNotExists(&dt.object, dt.bucket)
		assert.NotNil(dt.T(), job)
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
