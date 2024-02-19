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

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
)

// JobManager is responsible for maintaining, getting and removing file download
// jobs. It is created only once at the time of mounting.
type JobManager struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	// filePerm is passed to Job created by JobManager. It decides the
	// permission of file in cache created by Job.
	filePerm os.FileMode
	// dirPerm is passed to Job created by JobManager. dirPerm decides the
	// permission of directory at cache location created by Job.
	dirPerm os.FileMode
	// cacheDir is the path to directory where cache files should be created.
	cacheDir string
	// sequentialReadSizeMb is passed to Job created by JobManager, and it decides
	// the size of GCS read requests by Job at the time of downloading object to
	// file in cache.
	sequentialReadSizeMb int32
	fileInfoCache        *lru.Cache

	/////////////////////////
	// Mutable state
	/////////////////////////

	// jobs contains the reference to Job for a given object path. Object path is
	// concatenation of bucket name, "/", and object name. e.g. object path for an
	// object named "a/b/foo.txt" in bucket named "test_bucket" would be
	// "test_bucket/a/b/foo.txt"
	jobs map[string]*Job
	mu   locker.Locker
}

func NewJobManager(fileInfoCache *lru.Cache, filePerm os.FileMode, dirPerm os.FileMode, cacheDir string, sequentialReadSizeMb int32) (jm *JobManager) {
	jm = &JobManager{fileInfoCache: fileInfoCache, filePerm: filePerm,
		dirPerm: dirPerm, cacheDir: cacheDir, sequentialReadSizeMb: sequentialReadSizeMb}
	jm.mu = locker.New("JobManager", func() {})
	jm.jobs = make(map[string]*Job)
	return
}

// removeJob is a helper function to remove downloader.Job for given object and
// bucket from jm.jobs if present. It is passed as callback function to job so
// that job can remove itself after completion/failure/invalidation.
//
// Acquires and releases Lock(jm.mu)
func (jm *JobManager) removeJob(objectName string, bucketName string) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	objectPath := util.GetObjectPath(bucketName, objectName)
	delete(jm.jobs, objectPath)
}

// CreateJobIfNotExists creates and returns downloader.Job for given object and bucket.
// If there is already an existing job then this method returns that.
//
// Acquires and releases Lock(jm.mu)
func (jm *JobManager) CreateJobIfNotExists(object *gcs.MinObject, bucket gcs.Bucket) (job *Job) {
	objectPath := util.GetObjectPath(bucket.Name(), object.Name)
	jm.mu.Lock()
	defer jm.mu.Unlock()
	job, ok := jm.jobs[objectPath]
	if ok {
		return job
	}
	downloadPath := util.GetDownloadPath(jm.cacheDir, objectPath)
	fileSpec := data.FileSpec{Path: downloadPath, FilePerm: jm.filePerm, DirPerm: jm.dirPerm}
	// Pass call back function to Job. When this callback function is called, it
	// removes the job reference from jobs map.
	removeJobCallback := func() {
		jm.removeJob(object.Name, bucket.Name())
	}
	job = NewJob(object, bucket, jm.fileInfoCache, jm.sequentialReadSizeMb, fileSpec, removeJobCallback)
	jm.jobs[objectPath] = job
	return job
}

// GetJob returns downloader.Job for given object and bucket if present. If the
// job is not present, it returns nil.
//
// Acquires and releases Lock(jm.mu)
func (jm *JobManager) GetJob(objectName string, bucketName string) *Job {
	objectPath := util.GetObjectPath(bucketName, objectName)
	jm.mu.Lock()
	defer jm.mu.Unlock()
	job := jm.jobs[objectPath]
	return job
}

// InvalidateAndRemoveJob invalidates downloader.Job for given object and bucket.
// If there is no existing job present then this method does nothing.
// Note: Invalidating a job also removes job from jm.jobs map.
//
// Acquires and releases Lock(jm.mu)
func (jm *JobManager) InvalidateAndRemoveJob(objectName string, bucketName string) {
	objectPath := util.GetObjectPath(bucketName, objectName)
	jm.mu.Lock()
	job, ok := jm.jobs[objectPath]
	// Release the lock while calling downloader.Job.Invalidate to avoid deadlock
	// as the job calls removeJobCallback in the end which requires Lock(jm.mu).
	jm.mu.Unlock()
	if ok {
		job.Invalidate()
	}
}

// Destroy invalidates and deletes all the jobs that job manager is managing.
//
// Acquires and releases Lock(jm.mu)
func (jm *JobManager) Destroy() {
	// Get all jobs
	jm.mu.Lock()
	jobs := make([]*Job, 0, len(jm.jobs))
	for _, job := range jm.jobs {
		jobs = append(jobs, job)
	}
	jm.mu.Unlock()

	// Invalidate all the jobs which internally also removes from jm.jobs.
	for _, job := range jobs {
		job.Invalidate()
	}
}
