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
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	cacheutil "github.com/googlecloudplatform/gcsfuse/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/internal/monitor"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/internal/util"
	"golang.org/x/net/context"
)

type jobStatusName string

const (
	NotStarted  jobStatusName = "NotStarted"
	Downloading jobStatusName = "Downloading"
	Completed   jobStatusName = "Completed"
	Failed      jobStatusName = "Failed"
	Invalid     jobStatusName = "Invalid"
)

const ReadChunkSize = 8 * cacheutil.MiB

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

	// status represents the current status of Job.
	// status.Offset means data in cache is present in range [0, status.offset)
	status JobStatus

	// subscribers is list of subscribers waiting on async download.
	//
	// INVARIANT: Each element is of type jobSubscriber
	subscribers list.List

	// Context & its CancelFunc for cancelling async download in progress.
	// These properties are non-nil only while the async job is running.
	cancelCtx  context.Context
	cancelFunc context.CancelFunc

	// doneCh for waiting for cancellation of async download in progress.
	doneCh chan struct{}

	// removeJobCallback is a callback function to remove job from JobManager. It
	// is responsibility of JobManager to pass this function.
	removeJobCallback func()

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
	sequentialReadSizeMb int32, fileSpec data.FileSpec, removeJobCallback func()) (job *Job) {
	job = &Job{
		object:               object,
		bucket:               bucket,
		fileInfoCache:        fileInfoCache,
		sequentialReadSizeMb: sequentialReadSizeMb,
		fileSpec:             fileSpec,
		removeJobCallback:    removeJobCallback,
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
	job.status = JobStatus{NotStarted, nil, 0}
	job.subscribers = list.List{}
	job.doneCh = make(chan struct{})
}

// cancel is helper function to cancel the in-progress job.downloadAsync goroutine.
// This call is blocking until the job.downloadAsync terminates. Also, it should
// only be called when job.downloadAsync goroutine is running.
//
// Requires and releases LOCK(job.mu)
func (job *Job) cancel() {
	// job.cancelFunc = nil means that the async job has already
	// completed/cancelled/failed.
	if job.cancelFunc == nil {
		job.mu.Unlock()
		return
	}

	job.cancelFunc()
	// Unlock job.mu for the job.downloadAsync to terminate as it may require lock
	// to complete the inflight operations. In case, failure/update of offset
	// occurs while performing the inflight operations, the subscribers will be
	// notified with the failure/update and that is fine because if it is a failure
	// then subscribers should anyway be handling that and if it is an update, then
	// that's a successful update, so subscriber can attempt to read from file in
	// cache.
	job.mu.Unlock()
	// Wait for cancellation of job.downloadAsync.
	<-job.doneCh
}

// Invalidate invalidates the download job i.e. changes the state to Invalid.
// If the async download is in progress, this function cancels that. The caller
// should not read from the file in cache if job is in Invalid state.
// Note: job.removeJobCallback function is also executed as part of invalidation.
//
// Acquires and releases LOCK(job.mu)
func (job *Job) Invalidate() {
	job.mu.Lock()
	if job.status.Name == Downloading {
		job.status.Name = Invalid
		job.cancel()

		// Lock again to execute common notification logic.
		job.mu.Lock()
	}
	defer job.mu.Unlock()
	job.status.Name = Invalid
	logger.Tracef("Job:%p (%s:/%s) is no longer valid.", job, job.bucket.Name(), job.object.Name)
	if job.removeJobCallback != nil {
		job.removeJobCallback()
		job.removeJobCallback = nil
	}
	job.notifySubscribers()
}

// subscribe adds subscriber for download job and returns channel which is
// notified when the download is completed at least till the subscribed offset
// or in case of failure and invalidation.
//
// Not concurrency safe and requires LOCK(job.mu)
func (job *Job) subscribe(subscribedOffset int64) (notificationC <-chan JobStatus) {
	subscriberC := make(chan JobStatus, 1)
	job.subscribers.PushBack(jobSubscriber{subscriberC, subscribedOffset})
	return subscriberC
}

// notifySubscribers notifies all the subscribers of download job in case of
// failure or invalidation or completion till the subscribed offset.
//
// Not concurrency safe and requires LOCK(job.mu)
func (job *Job) notifySubscribers() {
	var nextSubItr *list.Element
	for subItr := job.subscribers.Front(); subItr != nil; subItr = nextSubItr {
		subItrValue := subItr.Value.(jobSubscriber)
		nextSubItr = subItr.Next()
		if job.status.Name == Failed || job.status.Name == Invalid || job.status.Offset >= subItrValue.subscribedOffset {
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
	logger.Errorf("Job:%p (%s:/%s) failed with: %v", job, job.bucket.Name(), job.object.Name, downloadErr)
	job.mu.Lock()
	job.status.Err = downloadErr
	job.status.Name = Failed
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
		err = fmt.Errorf("updateFileInfoCache: error while creating fileInfoKeyName for bucket %s and object %s %w",
			fileInfoKey.BucketName, fileInfoKey.ObjectName, err)
		return
	}

	updatedFileInfo := data.FileInfo{
		Key: fileInfoKey, ObjectGeneration: job.object.Generation,
		FileSize: job.object.Size, Offset: uint64(job.status.Offset),
	}

	logger.Tracef("Job:%p (%s:/%s) downloaded till %v offset.", job, job.bucket.Name(), job.object.Name, job.status.Offset)
	err = job.fileInfoCache.UpdateWithoutChangingOrder(fileInfoKeyName, updatedFileInfo)
	if err != nil {
		err = fmt.Errorf("updateFileInfoCache: error while inserting into fileInfoCache %s: %w", updatedFileInfo.Key, err)
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
	// Close the job.doneCh, clear the cancelFunc & cancelCtx and call the
	// remove job callback function in any case - completion/failure.
	defer func() {
		job.cancelFunc()
		close(job.doneCh)

		job.mu.Lock()
		if job.removeJobCallback != nil {
			job.removeJobCallback()
			job.removeJobCallback = nil
		}
		job.cancelCtx, job.cancelFunc = nil, nil
		job.mu.Unlock()
	}()

	// Create, open and truncate cache file for writing object into it.
	cacheFile, err := cacheutil.CreateFile(job.fileSpec, os.O_TRUNC|os.O_WRONLY)
	if err != nil {
		err = fmt.Errorf("downloadObjectAsync: error in creating cache file: %w", err)
		job.failWhileDownloading(err)
		return
	}
	defer func() {
		err = cacheFile.Close()
		if err != nil {
			err = fmt.Errorf("downloadObjectAsync: error while closing cache file: %w", err)
			job.failWhileDownloading(err)
		}
	}()

	notifyInvalid := func() {
		job.mu.Lock()
		job.status.Name = Invalid
		job.notifySubscribers()
		job.mu.Unlock()
	}

	var newReader io.ReadCloser
	var start, end, sequentialReadSize, newReaderLimit int64
	end = int64(job.object.Size)
	sequentialReadSize = int64(job.sequentialReadSizeMb) * cacheutil.MiB

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
						// Context is canceled when job.cancel is called at the time of
						// invalidation and hence caller should be notified as invalid.
						if errors.Is(err, context.Canceled) {
							notifyInvalid()
							return
						}
						err = fmt.Errorf("downloadObjectAsync: error in creating NewReader with start %d and limit %d: %w", start, newReaderLimit, err)
						job.failWhileDownloading(err)
						return
					}
					monitor.CaptureGCSReadMetrics(job.cancelCtx, util.Sequential, newReaderLimit-start)
				}

				maxRead := min(ReadChunkSize, newReaderLimit-start)
				_, err = cacheFile.Seek(start, 0)
				if err != nil {
					err = fmt.Errorf("downloadObjectAsync: error while seeking file handle, seek %d: %w", start, err)
					job.failWhileDownloading(err)
					return
				}

				// Copy the contents from NewReader to cache file.
				_, readErr := io.CopyN(cacheFile, newReader, maxRead)
				if readErr != nil {
					// Context is canceled when job.cancel is called at the time of
					// invalidation and hence caller should be notified as invalid.
					if errors.Is(readErr, context.Canceled) {
						notifyInvalid()
						return
					}
					err = fmt.Errorf("downloadObjectAsync: error at the time of copying content to cache file %w", readErr)
					job.failWhileDownloading(err)
					return
				}
				start += maxRead
				if start == newReaderLimit {
					newReader = nil
				}

				job.mu.Lock()
				job.status.Offset = start
				err = job.updateFileInfoCache()
				// Notify subscribers if file cache is updated.
				if err == nil {
					job.notifySubscribers()
				} else if strings.Contains(err.Error(), lru.EntryNotExistErrMsg) {
					// Download job expects entry in file info cache for the file it is
					// downloading. If the entry is deleted in between which is expected
					// to happen at the time of eviction, then the job should be
					// marked Invalid instead of Failed.
					job.status.Name = Invalid
					job.notifySubscribers()
					logger.Tracef("Job:%p (%s:/%s) is no longer valid due to absense of entry in file info cache.", job, job.bucket.Name(), job.object.Name)
					job.mu.Unlock()
					return
				}
				job.mu.Unlock()
				// Change status of job in case of error while updating file cache.
				if err != nil {
					job.failWhileDownloading(err)
					return
				}
			} else {
				job.mu.Lock()
				job.status.Name = Completed
				job.notifySubscribers()
				job.mu.Unlock()
				return
			}
		}
	}
}

// Download downloads object till the given offset and returns the status of
// job. If the object is already downloaded or there was failure in download,
// then it returns the job status. The caller shouldn't read data from file in
// cache if jobStatus is Failed or Invalid.
//
// Acquires and releases LOCK(job.mu)
func (job *Job) Download(ctx context.Context, offset int64, waitForDownload bool) (jobStatus JobStatus, err error) {
	job.mu.Lock()
	if int64(job.object.Size) < offset {
		defer job.mu.Unlock()
		err = fmt.Errorf(fmt.Sprintf("Download: the requested offset %d is greater than the size of object %d", offset, job.object.Size))
		return job.status, err
	}

	if job.status.Name == Completed {
		defer job.mu.Unlock()
		return job.status, nil
	} else if job.status.Name == NotStarted {
		// Start the async download
		job.status.Name = Downloading
		job.cancelCtx, job.cancelFunc = context.WithCancel(context.Background())
		go job.downloadObjectAsync()
	} else if job.status.Name == Failed || job.status.Name == Invalid || job.status.Offset >= offset {
		defer job.mu.Unlock()
		return job.status, nil
	}

	if !waitForDownload {
		defer job.mu.Unlock()
		return job.status, nil
	}

	// Subscribe to the given offset.
	notificationC := job.subscribe(offset)
	// Lock is not required when the subscriber is waiting for async download job
	// to download the requested contents.
	job.mu.Unlock()

	// Wait till subscriber is notified or the context is cancelled.
	select {
	case <-ctx.Done():
		err = fmt.Errorf("Download: %w", ctx.Err())
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
