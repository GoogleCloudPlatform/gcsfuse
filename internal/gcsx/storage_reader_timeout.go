package gcsx

import (
	"errors"
	"fmt"
	"time"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"golang.org/x/net/context"
)

/*
*
A wrapper over gcs.StorageReader, which encapsulates the logic of closing the inactive
reader after a given timeout.

Important notes for caller:
(a) Calling Read() after Close() will lead to unexpected behavior.
(b) Returns DontUseErr if you want to use with zero timeout, instead use the direct reader.
(c) Reader is completely thread-safe, that means multiple go routines can use the same reader.

Handling race scenario (internal implementation):
(a) closeReaderDueToInactivity call happens because of inactivity.
(b) At the same time, new Read() call comes.
(c) Two options:
  - Read() execution happens before closeReaderDueToInactivity().
  - timer.Stop() will return false.
  - timer.Reset() will creates a new schedules of closeReaderDueToInactivity().
  - Basically, two instance of closeReaderDueToInactivity - one will be executed just
    after the Read() call, and other after timeout. We can discard the execution of one
    by keeping a variable to discard.
  - Read() execution happens after closeReaderDueToInactivity().
  - No issues, closeReaderDueToInactivity will close the reader and Read() will create a new one.
*/
type timedStorageReader struct {
	object    *gcs.MinObject
	bucket    gcs.Bucket
	gcsReader gcs.StorageReader

	seen  int64
	start int64
	end   int64

	readHandle    []byte
	parentContext context.Context

	mu       locker.Locker
	isActive bool
	stopChan chan struct{}
}

var (
	DontUseErr = errors.New("DontUseErr")
)

func NewStorageReaderWithInactiveTimeout(ctx context.Context, bucket gcs.Bucket, object *gcs.MinObject, readHandle []byte, start int64, end int64, timeout time.Duration) (gcs.StorageReader, error) {

	if timeout == time.Duration(0) {
		return nil, DontUseErr
	}

	tsr := &timedStorageReader{
		object:        object,
		bucket:        bucket,
		start:         start,
		end:           end,
		parentContext: ctx,
		readHandle:    readHandle,
		mu:            locker.New("StorageReaderWithInactiveTimeout: "+object.Name, func() {}),
		isActive:      false,
		stopChan:      make(chan struct{}),
	}

	var err error
	tsr.gcsReader, err = bucket.NewReaderWithReadHandle(
		ctx,
		&gcs.ReadObjectRequest{
			Name:       object.Name,
			Generation: object.Generation,
			Range: &gcs.ByteRange{
				Start: uint64(start),
				Limit: uint64(end),
			},
			ReadCompressed: object.HasContentEncodingGzip(),
			ReadHandle:     readHandle,
		})

	go tsr.monitor(timeout)
	return tsr, err
}

func (tsr *timedStorageReader) Read(p []byte) (n int, err error) {
	tsr.mu.Lock()
	defer tsr.mu.Unlock()

	tsr.isActive = true

	if tsr.gcsReader == nil {
		tsr.gcsReader, err = tsr.bucket.NewReaderWithReadHandle(
			tsr.parentContext,
			&gcs.ReadObjectRequest{
				Name:       tsr.object.Name,
				Generation: tsr.object.Generation,
				Range: &gcs.ByteRange{
					Start: uint64(tsr.start + tsr.seen),
					Limit: uint64(tsr.end),
				},
				ReadCompressed: tsr.object.HasContentEncodingGzip(),
				ReadHandle:     tsr.readHandle,
			})

		if err != nil {
			err = fmt.Errorf("NewReaderWithReadHandle: %w", err)
			return
		}
	}

	n, err = tsr.gcsReader.Read(p)
	tsr.seen += int64(n)
	return
}

func (tsr *timedStorageReader) Close() (err error) {
	tsr.mu.Lock()
	defer tsr.mu.Unlock()

	close(tsr.stopChan) // Close background monitoring routine.
	err = tsr.gcsReader.Close()
	if err != nil {
		return fmt.Errorf("close reader: %w", err)
	}
	return
}

func (tsr *timedStorageReader) ReadHandle() (rh storagev2.ReadHandle) {
	tsr.mu.Lock()
	defer tsr.mu.Unlock()

	if tsr.gcsReader == nil {
		return tsr.readHandle
	}

	return tsr.gcsReader.ReadHandle()
}

func (tsr *timedStorageReader) monitor(timeout time.Duration) {
	var done bool
	timer := time.After(timeout)
	for !done {
		select {
		case <-timer:
			tsr.mu.Lock()
			if tsr.isActive {
				tsr.isActive = false
				timer = time.After(timeout)
			} else {
				if tsr.gcsReader != nil {
					logger.Infof("Closing the reader (%s) due to inactivity for %0.1fs.\n", tsr.object.Name, timeout.Seconds())
					tsr.readHandle = tsr.gcsReader.ReadHandle()
					err := tsr.gcsReader.Close()
					if err != nil {
						logger.Warnf("error while closing reader: %v", err)
					}
					tsr.gcsReader = nil
				}
			}
			tsr.mu.Unlock()
		case <-tsr.parentContext.Done():
			done = true

		case <-tsr.stopChan:
			done = true
		}
	}
}
