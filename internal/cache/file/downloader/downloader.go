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

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
)

// getObjectPath gives object path which is concatenation of bucket and object
// name separated by "/".
func getObjectPath(bucketName string, objectName string) string {
	return path.Join(bucketName, objectName)
}

// JobManager is responsible for maintaining, getting and removing file download
// jobs.
type JobManager struct {
	fileInfoCache        *lru.Cache
	perm                 os.FileMode
	cacheLocation        string
	sequentialReadSizeMb int32

	jobs map[string]*Job
	mu   locker.Locker
}

func NewJobManager(fileInfoCache *lru.Cache, perm os.FileMode, cacheLocation string, sequentialReadSizeMb int32) (jm *JobManager) {
	jm = &JobManager{fileInfoCache: fileInfoCache, perm: perm,
		cacheLocation: cacheLocation, sequentialReadSizeMb: sequentialReadSizeMb}
	jm.mu = locker.New("JobManager", func() {})
	jm.jobs = make(map[string]*Job)
	return
}

// getDownloadPath gives file path to file in cache for given object path.
func (jm *JobManager) getDownloadPath(objectPath string) string {
	return path.Join(jm.cacheLocation, objectPath)
}

// GetJob gives downloader.Job for given object and bucket. If there is no
// existing job then this method creates one.
//
// Acquires and releases Lock(jm.mu)
func (jm *JobManager) GetJob(object *gcs.MinObject, bucket gcs.Bucket) (job *Job) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	objectPath := getObjectPath(bucket.Name(), object.Name)
	job, ok := jm.jobs[objectPath]
	if !ok {
		downloadPath := jm.getDownloadPath(objectPath)
		fileSpec := data.FileSpec{Path: downloadPath, Perm: jm.perm}
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
	objectPath := getObjectPath(bucketName, objectName)
	job, ok := jm.jobs[objectPath]
	if ok {
		job.Invalidate()
		delete(jm.jobs, objectPath)
	}
}
