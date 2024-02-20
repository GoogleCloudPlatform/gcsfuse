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
// jobs.
type JobManager struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	// filePerm is passed to Job created by JobManager. filePerm decides the
	// permission of file in cache created by Job.
	filePerm os.FileMode
	// dirPerm is passed to Job created by JobManager. dirPerm decides the
	// permission of directory at cache location created by Job.
	dirPerm os.FileMode
	// cacheLocation is the path to directory where cache files should be created.
	cacheLocation string
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

func NewJobManager(fileInfoCache *lru.Cache, filePerm os.FileMode, dirPerm os.FileMode, cacheLocation string, sequentialReadSizeMb int32) (jm *JobManager) {
	jm = &JobManager{fileInfoCache: fileInfoCache, filePerm: filePerm,
		dirPerm: dirPerm, cacheLocation: cacheLocation, sequentialReadSizeMb: sequentialReadSizeMb}
	jm.mu = locker.New("JobManager", func() {})
	jm.jobs = make(map[string]*Job)
	return
}

// GetJob gives downloader.Job for given object and bucket. If there is no
// existing job then this method creates one.
//
// Acquires and releases Lock(jm.mu)
func (jm *JobManager) GetJob(object *gcs.MinObject, bucket gcs.Bucket) (job *Job) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	objectPath := util.GetObjectPath(bucket.Name(), object.Name)
	job, ok := jm.jobs[objectPath]
	if !ok {
		downloadPath := util.GetDownloadPath(jm.cacheLocation, objectPath)
		fileSpec := data.FileSpec{Path: downloadPath, FilePerm: jm.filePerm, DirPerm: jm.dirPerm}
		job = NewJob(object, bucket, jm.fileInfoCache, jm.sequentialReadSizeMb, fileSpec)
		jm.jobs[objectPath] = job
	}
	return job
}

// RemoveJob removes downloader.Job for given object and bucket. If there is no
// existing job present then this method does nothing. Also, the job is
// first invalidated before removing.
//
// Acquires and releases Lock(jm.mu)
func (jm *JobManager) RemoveJob(objectName string, bucketName string) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	objectPath := util.GetObjectPath(bucketName, objectName)
	job, ok := jm.jobs[objectPath]
	if ok {
		job.Invalidate()
		delete(jm.jobs, objectPath)
	}
}

// Destroy invalidates and deletes all the jobs that job manager is managing.
//
// Acquires and releases Lock(jm.mu)
func (jm *JobManager) Destroy() {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	for objectPath, job := range jm.jobs {
		job.Invalidate()
		delete(jm.jobs, objectPath)
	}
}
