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
	"container/list"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"syscall"

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/cache/lru"
	cacheutil "github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/util"
	"golang.org/x/net/context"
	"golang.org/x/sync/semaphore"
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
	fileCacheConfig      *cfg.FileCacheConfig

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
	// This semaphore is shared across all jobs spawned by the job manager and is
	// used to limit the download concurrency.
	maxParallelismSem *semaphore.Weighted

	// Channel which is used by goroutines to know which ranges need to be
	// downloaded when parallel download is enabled.
	rangeChan chan data.ObjectRange

	metricsHandle common.MetricHandle
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

func NewJob(
	object *gcs.MinObject,
	bucket gcs.Bucket,
	fileInfoCache *lru.Cache,
	sequentialReadSizeMb int32,
	fileSpec data.FileSpec,
	removeJobCallback func(),
	fileCacheConfig *cfg.FileCacheConfig,
	maxParallelismSem *semaphore.Weighted,
	metricHandle common.MetricHandle,
) (job *Job) {
	job = &Job{
		object:               object,
		bucket:               bucket,
		fileInfoCache:        fileInfoCache,
		sequentialReadSizeMb: sequentialReadSizeMb,
		fileSpec:             fileSpec,
		removeJobCallback:    removeJobCallback,
		fileCacheConfig:      fileCacheConfig,
		maxParallelismSem:    maxParallelismSem,
		metricsHandle:        metricHandle,
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

// updateStatusAndNotifySubscribers is a helper function which updates the
// status's name and err to given name and error and notifies subscribers.
//
// Acquires and releases Lock(job.mu)
func (job *Job) updateStatusAndNotifySubscribers(statusName jobStatusName, statusErr error) {
	if statusName == Failed {
		logger.Errorf("Job:%p (%s:/%s) Failed with error: %v", job, job.bucket.Name(), job.object.Name, statusErr)
	} else {
		logger.Tracef("Job:%p (%s:/%s) status changed to %v with error: %v", job, job.bucket.Name(), job.object.Name, statusName, statusErr)
	}
	job.mu.Lock()
	job.status.Err = statusErr
	job.status.Name = statusName
	job.notifySubscribers()
	job.mu.Unlock()
}

// updateStatusOffset updates the offset in job's status and in file info cache
// with the given offset. If the update is successful, this function also
// notify the subscribers.
// Not concurrency safe and requires LOCK(job.mu)
func (job *Job) updateStatusOffset(downloadedOffset int64) (err error) {
	fileInfoKey := data.FileInfoKey{
		BucketName: job.bucket.Name(),
		ObjectName: job.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		err = fmt.Errorf("updateStatusOffset: error while creating fileInfoKeyName for bucket %s and object %s %w",
			fileInfoKey.BucketName, fileInfoKey.ObjectName, err)
		return err
	}

	updatedFileInfo := data.FileInfo{
		Key: fileInfoKey, ObjectGeneration: job.object.Generation,
		FileSize: job.object.Size, Offset: uint64(downloadedOffset),
	}

	err = job.fileInfoCache.UpdateWithoutChangingOrder(fileInfoKeyName, updatedFileInfo)
	if err == nil {
		job.status.Offset = downloadedOffset
		// Notify subscribers if file cache is updated.
		logger.Tracef("Job:%p (%s:/%s) downloaded till %v offset.", job, job.bucket.Name(), job.object.Name, job.status.Offset)
		job.notifySubscribers()
		return err
	}

	err = fmt.Errorf("updateStatusOffset: error while updating offset: %v in fileInfoCache %s: %w", downloadedOffset, updatedFileInfo.Key, err)
	return err
}

// downloadObjectToFile downloads the backing object from GCS into the given
// file and updates the file info cache. It uses gcs.Bucket's NewReader method
// to download the object.
func (job *Job) downloadObjectToFile(cacheFile *os.File) (err error) {
	var newReader io.ReadCloser
	var start, end, sequentialReadSize, newReaderLimit int64
	end = int64(job.object.Size)
	sequentialReadSize = int64(job.sequentialReadSizeMb) * cacheutil.MiB

	// Each iteration of this for loop, reads ReadChunkSize size of range of the
	// backing object from reader into the file handle and updates the file info
	// cache. In case, reader is not present for reading, it creates a
	// gcs.Bucket's NewReader with size min(sequentialReadSize, object.Size).
	for start < end {
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
				err = fmt.Errorf("downloadObjectToFile: error in creating NewReader with start %d and limit %d: %w", start, newReaderLimit, err)
				return err
			}
			common.CaptureGCSReadMetrics(job.cancelCtx, job.metricsHandle, util.Sequential, newReaderLimit-start)
		}

		maxRead := min(ReadChunkSize, newReaderLimit-start)

		// Copy the contents from NewReader to cache file.
		offsetWriter := io.NewOffsetWriter(cacheFile, start)
		_, err = io.CopyN(offsetWriter, newReader, maxRead)
		if err != nil {
			err = fmt.Errorf("downloadObjectToFile: error at the time of copying content to cache file %w", err)
			return err
		}

		start += maxRead
		if start == newReaderLimit {
			// Reader is closed after the data has been read and the error from closure
			// is not reported as failure of async job, similar to how it's done for
			// foreground reads: https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/internal/gcsx/random_reader.go#L298.
			err = newReader.Close()
			if err != nil {
				logger.Warnf("Job:%p (%s:/%s) error while closing reader: %v", job, job.bucket.Name(), job.object.Name, err)
			}
			newReader = nil
		}

		job.mu.Lock()
		err = job.updateStatusOffset(start)
		job.mu.Unlock()
		if err != nil {
			return err
		}
	}
	return nil
}

// cleanUpDownloadAsyncJob is a helper function which performs clean up tasks
// for the async job and this should be called at the end of async job.
//
// Acquires and releases LOCK(job.mu)
func (job *Job) cleanUpDownloadAsyncJob() {
	// Close the job.doneCh, clear the cancelFunc & cancelCtx and call the
	// remove job callback function.
	job.cancelFunc()
	close(job.doneCh)

	job.mu.Lock()
	if job.removeJobCallback != nil {
		job.removeJobCallback()
		job.removeJobCallback = nil
	}
	job.cancelCtx, job.cancelFunc = nil, nil
	job.mu.Unlock()
}

// createCacheFile is a helper function which creates file in cache using
// appropriate open file flags.
func (job *Job) createCacheFile() (*os.File, error) {
	// Create, open and truncate cache file for writing object into it.
	openFileFlags := os.O_TRUNC | os.O_WRONLY
	var cacheFile *os.File
	var err error
	// Try using O_DIRECT while opening file when parallel downloads are enabled
	// and O_DIRECT use is not disabled.
	if job.fileCacheConfig.EnableParallelDownloads && job.fileCacheConfig.EnableODirect {
		cacheFile, err = cacheutil.CreateFile(job.fileSpec, openFileFlags|syscall.O_DIRECT)
		if errors.Is(err, fs.ErrInvalid) || errors.Is(err, syscall.EINVAL) {
			logger.Warnf("downloadObjectAsync: failure in opening file with O_DIRECT, falling back to without O_DIRECT")
			cacheFile, err = cacheutil.CreateFile(job.fileSpec, openFileFlags)
		}
	} else {
		cacheFile, err = cacheutil.CreateFile(job.fileSpec, openFileFlags)
	}

	return cacheFile, err
}

// downloadObjectAsync downloads the backing GCS object into a file as part of
// file cache using NewReader method of gcs.Bucket.
//
// Note: There can only be one async download running for a job at a time.
// Acquires and releases LOCK(job.mu)
func (job *Job) downloadObjectAsync() {
	// Cleanup the async job in all cases - completion/failure/invalidation.
	defer job.cleanUpDownloadAsyncJob()

	cacheFile, err := job.createCacheFile()
	if err != nil {
		err = fmt.Errorf("downloadObjectAsync: error in creating cache file: %w", err)
		job.handleError(err)
		return
	}
	defer func() {
		err = cacheFile.Close()
		if err != nil {
			err = fmt.Errorf("downloadObjectAsync: error while closing cache file: %w", err)
			job.handleError(err)
		}
	}()

	// Both parallel and non-parallel download functions support cancellation in
	// case of job's cancellation.
	if job.fileCacheConfig.EnableParallelDownloads {
		err = job.parallelDownloadObjectToFile(cacheFile)
	} else {
		err = job.downloadObjectToFile(cacheFile)
	}

	if err != nil {
		// Download job expects entry in file info cache for the file it is
		// downloading. If the entry is deleted in between which is expected
		// to happen at the time of eviction, then the job should be
		// marked Invalid instead of Failed.
		if strings.Contains(err.Error(), lru.EntryNotExistErrMsg) {
			job.updateStatusAndNotifySubscribers(Invalid, err)
			return
		}
		job.handleError(err)
		return
	}

	// Truncate as the parallel downloads can create file with size little higher
	// than the actual object size because writing with O_DIRECT happens in size
	// multiple of cacheutil.MinimumAlignSizeForWriting.
	err = cacheFile.Truncate(int64(job.object.Size))
	if err != nil {
		err = fmt.Errorf("downloadObjectAsync: error while truncating cache file: %w", err)
		job.handleError(err)
		return
	}

	err = job.validateCRC()
	if err != nil {
		job.handleError(err)
		return
	}

	job.updateStatusAndNotifySubscribers(Completed, err)
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

// Compares CRC32 of the downloaded file with the CRC32 from GCS object metadata.
// In case of mismatch deletes the file and corresponding entry from file cache.
func (job *Job) validateCRC() (err error) {
	if !job.fileCacheConfig.EnableCrc {
		return
	}

	crc32Val, err := cacheutil.CalculateFileCRC32(job.cancelCtx, job.fileSpec.Path)
	if err != nil {
		return
	}

	// If checksum matches, simply return.
	if *job.object.CRC32C == crc32Val {
		return nil
	}

	// If the checksum doesn't match there is an error in downloading the object contents.
	// Delete the file and corresponding key from fileInfoCache.
	err = fmt.Errorf("checksum mismatch detected. Actual: %d, expected: %d", crc32Val, *job.object.CRC32C)
	fileInfoKey := data.FileInfoKey{
		BucketName: job.bucket.Name(),
		ObjectName: job.object.Name,
	}

	fileInfoKeyName, keyErr := fileInfoKey.Key()
	if keyErr != nil {
		err = errors.Join(err, keyErr)
		return
	}

	job.fileInfoCache.Erase(fileInfoKeyName)
	removeErr := cacheutil.TruncateAndRemoveFile(job.fileSpec.Path)
	if removeErr != nil && !os.IsNotExist(removeErr) {
		err = errors.Join(err, removeErr)
	}

	return
}

// Performs different actions based on the type of error.
// For context.Canceled it marks the job as invalid and notifies subscribers.
// For other errors, marks the job as failed and notifies subscribers.
func (job *Job) handleError(err error) {
	// Context is canceled when job.cancel is called at the time of
	// invalidation and hence caller should be notified as invalid.
	if errors.Is(err, context.Canceled) {
		job.updateStatusAndNotifySubscribers(Invalid, err)
		return
	}

	job.updateStatusAndNotifySubscribers(Failed, err)
}

func (job *Job) IsParallelDownloadsEnabled() bool {
	if job.fileCacheConfig != nil && job.fileCacheConfig.EnableParallelDownloads {
		return true
	}
	return false
}
