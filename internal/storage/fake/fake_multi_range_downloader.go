// Copyright 2025 Google LLC
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

package fake

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/storageutil"
)

// This struct is an implementation of the gcs.MultiRangeDownloader interface.
type fakeMultiRangeDownloader struct {
	gcs.MultiRangeDownloader
	obj        *fakeObject
	wg         sync.WaitGroup
	err        error
	defaultErr error
	statusErr  error
	sleepTime  time.Duration // Sleep time to simulate real-world.
	shortRead  bool
}

func createFakeObject(obj *gcs.MinObject, data []byte) fakeObject {
	fullObj := storageutil.ConvertMinObjectToObject(obj)
	return fakeObject{
		metadata: *fullObj,
		data:     data,
	}
}

func NewFakeMultiRangeDownloader(obj *gcs.MinObject, data []byte) gcs.MultiRangeDownloader {
	return NewFakeMultiRangeDownloaderWithSleepAndDefaultError(obj, data, time.Millisecond, nil)
}

func NewFakeMultiRangeDownloaderWithShortRead(obj *gcs.MinObject, data []byte) gcs.MultiRangeDownloader {
	fakeObject := createFakeObject(obj, data)
	return &fakeMultiRangeDownloader{
		obj:       &fakeObject,
		shortRead: true,
	}
}

func NewFakeMultiRangeDownloaderWithSleep(obj *gcs.MinObject, data []byte, sleepTime time.Duration) gcs.MultiRangeDownloader {
	return NewFakeMultiRangeDownloaderWithSleepAndDefaultError(obj, data, sleepTime, nil)
}

func NewFakeMultiRangeDownloaderWithSleepAndDefaultError(obj *gcs.MinObject, data []byte, sleepTime time.Duration, err error) gcs.MultiRangeDownloader {
	fakeObj := createFakeObject(obj, data)
	return &fakeMultiRangeDownloader{
		obj:        &fakeObj,
		sleepTime:  sleepTime,
		defaultErr: err,
	}
}

func NewFakeMultiRangeDownloaderWithStatusError(obj *gcs.MinObject, data []byte, err error) gcs.MultiRangeDownloader {
	fakeObj := createFakeObject(obj, data)
	return &fakeMultiRangeDownloader{
		obj:        &fakeObj,
		sleepTime:  0,
		defaultErr: nil,
		statusErr:  err,
	}
}

func (fmrd *fakeMultiRangeDownloader) Add(output io.Writer, offset, length int64, callback func(int64, int64, error)) {
	if fmrd.defaultErr != nil {
		if callback != nil {
			callback(offset, 0, fmrd.defaultErr)
		}
		return
	}
	obj := fmrd.obj
	size := int64(len(obj.data))
	var err error
	// Apply input checks as defined at https://github.com/googleapis/go-storage-prelaunch/blob/a5db2abd53775941df67b3337eabaf8d00ef0762/storage/reader.go#L373 .
	if length < 0 {
		err = fmt.Errorf("length < 0")
	} else if offset > size {
		err = fmt.Errorf("out of range. offset (%v) > size of content (%v) of %s", offset, size, obj.metadata.Name)
	} else if offset <= -size {
		offset = 0
		length = size
	} else if offset < 0 {
		offset = size + offset
		length = min(length, size-offset)
	} else {
		length = min(length, size-offset)
	}
	if err != nil {
		// If inputs aren't correct, fail immediately and return callback.
		fmrd.err = err
		if callback != nil {
			callback(offset, 0, err)
		}
		return
	}

	// Record this additional goroutine.
	fmrd.wg.Add(1)

	go func() {
		// clear this goroutine from waitgroup.
		defer fmrd.wg.Done()

		if fmrd.shortRead {
			length /= 2
		}

		time.Sleep(fmrd.sleepTime)
		var n int
		n, err = output.Write(obj.data[offset : offset+length])
		if err != nil || int64(n) != length {
			err = fmt.Errorf("failed to write %v bytes to writer through multi-range-downloader, bytes written = %v, error = %v", length, n, err)
		}

		if fmrd.shortRead && err == nil {
			err = io.EOF
		}

		if callback != nil {
			callback(offset, int64(n), err)
		}
		// Don't clear pre-existing error in downloader.
		if fmrd.err != nil {
			fmrd.err = err
		}
	}()
}

func (fmrd *fakeMultiRangeDownloader) Close() error {
	fmrd.Wait()
	return fmrd.err
}

func (fmrd *fakeMultiRangeDownloader) Wait() {
	fmrd.wg.Wait()
}

func (fmrd *fakeMultiRangeDownloader) Error() error {
	return fmrd.statusErr
}

func (fmrd *fakeMultiRangeDownloader) GetHandle() []byte {
	return nil
}
