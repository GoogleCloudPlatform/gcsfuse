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
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"golang.org/x/net/context"
)

type jobStatusName string

const (
	NOT_STARTED jobStatusName = "NOT_STARTED"
	DOWNLOADING jobStatusName = "DOWNLOADING"
	COMPLETED   jobStatusName = "COMPLETED"
	FAILED      jobStatusName = "FAILED"
	CANCELLED   jobStatusName = "CANCELLED"
	INVALID     jobStatusName = "INVALID"
)

const ReadChunkSize = 8 * util.MiB

// Job downloads the requested object from GCS into the specified local file
// path with given permissions and ownership.
type Job struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	object               *gcs.MinObject
	bucket               gcs.Bucket
	fileInfoCache        *lru.Cache
	sequentialReadSizeMb int32
	fileSpec             data.FileSpec

	/////////////////////////
	// Mutable state
	/////////////////////////

	// Represents the current status of Job.
	// status.Offset means data in cache is present in range [0, status.offset)
	status JobStatus

	// list of subscribers waiting on async download.
	//
	// INVARIANT: Each element is of type jobSubscriber
	subscribers list.List

	// Context & its CancelFunc for cancelling async download in progress.
	cancelCtx  context.Context
	cancelFunc context.CancelFunc

	mu locker.Locker
}

// JobStatus represents the status of job.
type JobStatus struct {
	Name   jobStatusName
	Err    error
	Offset int64
}

// jobSubscriber represents a subscriber waiting on async download of job to
// complete downloading at least till the subscribed offset.
type jobSubscriber struct {
	notificationC    chan<- JobStatus
	subscribedOffset int64
}

func NewJob(object *gcs.MinObject, bucket gcs.Bucket, fileInfoCache *lru.Cache,
	sequentialReadSizeMb int32, fileSpec data.FileSpec) (job *Job) {
	job = &Job{
		object:               object,
		bucket:               bucket,
		fileInfoCache:        fileInfoCache,
		sequentialReadSizeMb: sequentialReadSizeMb,
		fileSpec:             fileSpec,
	}
	job.mu = locker.New("Job-"+fileSpec.Path, job.checkInvariants)
	job.init()
	return
}

// checkInvariants panic if any internal invariants have been violated.
func (job *Job) checkInvariants() {
	// INVARIANT: Each subscriber is of type jobSubscriber
	for e := job.subscribers.Front(); e != nil; e = e.Next() {
		switch e.Value.(type) {
		case jobSubscriber:
		default:
			panic(fmt.Sprintf("Unexpected element type: %v", reflect.TypeOf(e.Value)))
		}
	}
}

// init initializes the mutable members of Job corresponding to not started
// state.
func (job *Job) init() {
	job.status = JobStatus{NOT_STARTED, nil, 0}
	job.subscribers = list.List{}
	job.cancelCtx, job.cancelFunc = context.WithCancel(context.Background())
}

// Cancel changes the state of job to cancelled and cancels the async download
// job if there. Also, notifies the subscribers of job if any.
//
// Acquires and releases LOCK(job.mu)
func (job *Job) Cancel() {
	job.mu.Lock()
	defer job.mu.Unlock()
	if job.status.Name == DOWNLOADING || job.status.Name == NOT_STARTED {
		job.cancelFunc()
		job.status.Name = CANCELLED
		job.notifySubscribers()
	}
}

// Invalidate invalidates the download job i.e. changes the state to INVALID.
// If the async download is in progress, this function cancels that. The caller
// should not read from the file in cache if job is in INVALID state.
//
// Acquires and releases LOCK(job.mu)
func (job *Job) Invalidate() {
	job.mu.Lock()
	defer job.mu.Unlock()
	if job.status.Name == DOWNLOADING {
		job.cancelFunc()
	}
	job.status.Name = INVALID
	job.notifySubscribers()
}

// subscribe adds subscriber for download job and returns channel which is
// notified when the download is completed at least till the subscribed offset
// or in case of failure and cancellation.
//
// Not concurrency safe and requires LOCK(job.mu)
func (job *Job) subscribe(subscribedOffset int64) (notificationC <-chan JobStatus) {
	subscriberC := make(chan JobStatus, 1)
	job.subscribers.PushBack(jobSubscriber{subscriberC, subscribedOffset})
	return subscriberC
}

// notifySubscribers notifies all the subscribers of download job in case of
// error/cancellation or when download is completed till the subscribed offset.
//
// Not concurrency safe and requires LOCK(job.mu)
func (job *Job) notifySubscribers() {
	var nextSubItr *list.Element
	for subItr := job.subscribers.Front(); subItr != nil; subItr = nextSubItr {
		subItrValue := subItr.Value.(jobSubscriber)
		nextSubItr = subItr.Next()
		if job.status.Name == FAILED || job.status.Name == CANCELLED || job.status.Offset >= subItrValue.subscribedOffset {
			subItrValue.notificationC <- job.status
			close(subItrValue.notificationC)
			job.subscribers.Remove(subItr)
		}
	}
}

// failWhileDownloading changes the status of job to failed and notifies
// subscribers about the download error.
//
// Acquires and releases LOCK(job.mu)
func (job *Job) failWhileDownloading(downloadErr error) {
	job.mu.Lock()
	job.status.Err = downloadErr
	job.status.Name = FAILED
	job.notifySubscribers()
	job.mu.Unlock()
}

// updateFileInfoCache updates the file info cache with latest offset downloaded
// by job. Returns error in case of failure.
//
// Not concurrency safe and requires LOCK(job.mu)
func (job *Job) updateFileInfoCache() (err error) {
	fileInfoKey := data.FileInfoKey{
		BucketName: job.bucket.Name(),
		ObjectName: job.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		err = fmt.Errorf(fmt.Sprintf("init: error while calling FileInfoKey.Key() for bucket %s and object %s %v",
			fileInfoKey.BucketName, fileInfoKey.ObjectName, err))
		return
	}

	updatedFileInfo := data.FileInfo{
		Key: fileInfoKey, ObjectGeneration: job.object.Generation,
		FileSize: job.object.Size, Offset: uint64(job.status.Offset),
	}

	// To-Do(raj-prince): We should not call normal insert here as that internally
	// changes the LRU element which is undesirable given this is not user access.
	_, err = job.fileInfoCache.Insert(fileInfoKeyName, updatedFileInfo)
	if err != nil {
		err = fmt.Errorf(fmt.Sprintf("error while inserting updatedFileInfo to the FileInfoCache %s: %v", updatedFileInfo.Key, err))
		return
	}
	return
}

// downloadObjectAsync downloads the backing GCS object into a file as part of
// file cache using NewReader method of gcs.Bucket.
//
// Note: There can only be one async download running for a job at a time.
// Acquires and releases LOCK(job.mu)
func (job *Job) downloadObjectAsync() {
	// Create and open cache file for writing object into it.
	cacheFile, err := util.CreateFile(job.fileSpec, os.O_WRONLY)
	if err != nil {
		err = fmt.Errorf("downloadObjectAsync: error in creating cache file: %v", err)
		job.failWhileDownloading(err)
		return
	}
	defer func(cacheFile *os.File) {
		err = cacheFile.Close()
		if err != nil {
			err = fmt.Errorf("downloadObjectAsync: error while closing cache file: %v", err)
			job.failWhileDownloading(err)
		}
	}(cacheFile)

	var newReader io.ReadCloser
	var start, end, sequentialReadSize, newReaderLimit int64
	end = int64(job.object.Size)
	sequentialReadSize = int64(job.sequentialReadSizeMb) * util.MiB

	for {
		select {
		case <-job.cancelCtx.Done():
			return
		default:
			if start < end {
				if newReader == nil {
					newReaderLimit = min(start+sequentialReadSize, end)
					newReader, err = job.bucket.NewReader(
						job.cancelCtx,
						&gcs.ReadObjectRequest{
							Name:       job.object.Name,
							Generation: job.object.Generation,
							Range: &gcs.ByteRange{
								Start: uint64(start),
								Limit: uint64(newReaderLimit),
							},
							ReadCompressed: job.object.HasContentEncodingGzip(),
						})
					if err != nil {
						err = fmt.Errorf(fmt.Sprintf("downloadObjectAsync: error in creating NewReader with start %d and limit %d: %v", start, newReaderLimit, err))
						job.failWhileDownloading(err)
						return
					}
				}

				maxRead := min(end-start, ReadChunkSize)
				_, err = cacheFile.Seek(start, 0)
				if err != nil {
					err = fmt.Errorf(fmt.Sprintf("downloadObjectAsync: error while seeking file handle, seek %d: %v", start, err))
					job.failWhileDownloading(err)
					return
				}

				// copy the contents from NewReader to cache file.
				_, readErr := io.CopyN(cacheFile, newReader, maxRead)
				if readErr != nil && readErr != io.EOF {
					err = fmt.Errorf("downloadObjectAsync: error at the time of copying content to cache file %v", readErr)
					job.failWhileDownloading(err)
					return
				}
				start += maxRead
				if readErr == io.EOF {
					newReader = nil
				}

				job.mu.Lock()
				job.status.Offset = start
				err = job.updateFileInfoCache()
				// Notify subscribers if file cache is updated.
				if err == nil {
					job.notifySubscribers()
				}
				job.mu.Unlock()
				// change status of job in case of error while updating file cache.
				if err != nil {
					job.failWhileDownloading(err)
					return
				}
			} else {
				job.mu.Lock()
				job.status.Name = COMPLETED
				job.notifySubscribers()
				job.mu.Unlock()
				return
			}
		}
	}
}

// Download downloads object till the given offset and returns the status of
// job. If the object is already downloaded or there was failure/cancellation in
// download, then it returns the job status. The caller shouldn't read data
// from file in cache if jobStatus is FAILED or INVALID.
//
// Acquires and releases LOCK(job.mu)
func (job *Job) Download(ctx context.Context, offset int64, waitForDownload bool) (jobStatus JobStatus, err error) {
	job.mu.Lock()
	if int64(job.object.Size) < offset {
		defer job.mu.Unlock()
		err = fmt.Errorf(fmt.Sprintf("Download: the requested offset %d is greater than the size of object %d", offset, job.object.Size))
		return job.status, err
	}

	if job.status.Name == COMPLETED {
		defer job.mu.Unlock()
		return job.status, nil
	} else if job.status.Name == NOT_STARTED {
		// start the async download
		job.status.Name = DOWNLOADING
		go job.downloadObjectAsync()
	} else if job.status.Name == FAILED || job.status.Name == CANCELLED || job.status.Name == INVALID || job.status.Offset >= offset {
		defer job.mu.Unlock()
		return job.status, nil
	}

	if !waitForDownload {
		defer job.mu.Unlock()
		return job.status, nil
	}

	// subscribe to an offset.
	notificationC := job.subscribe(offset)
	// lock is not required when the subscriber is waiting for async download job.
	job.mu.Unlock()

	// Wait till subscriber is notified by async job or the async job is cancelled
	select {
	case <-ctx.Done():
		err = fmt.Errorf(fmt.Sprintf("Download: %v", ctx.Err()))
	case jobStatus = <-notificationC:
	}
	return
}

// GetStatus returns the status of download job.
//
// Acquires and releases LOCK(job.mu)
func (job *Job) GetStatus() JobStatus {
	job.mu.Lock()
	defer job.mu.Unlock()
	return job.status
}
