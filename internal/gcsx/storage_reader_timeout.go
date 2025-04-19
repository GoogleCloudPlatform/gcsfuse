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
type storageReaderWithInactiveTimeout struct {
	object    *gcs.MinObject
	bucket    gcs.Bucket
	gcsReader gcs.StorageReader

	seen  int64
	start int64
	end   int64

	readHandle    []byte
	parentContext context.Context
	timer         *time.Timer
	timeDuration  time.Duration

	discardInactivityCallbackOnce bool

	mu locker.Locker
}

var (
	DontUseErr = errors.New("DontUseErr")
)

func NewStorageReaderWithInactiveTimeout(ctx context.Context, bucket gcs.Bucket, object *gcs.MinObject, readHandle []byte, start int64, end int64, timeout time.Duration) (gcs.StorageReader, error) {

	if timeout == time.Duration(0) {
		return nil, DontUseErr
	}

	srwit := &storageReaderWithInactiveTimeout{
		object:        object,
		bucket:        bucket,
		start:         start,
		end:           end,
		parentContext: ctx,
		readHandle:    readHandle,
		timeDuration:  timeout,
		mu:            locker.New("StorageReaderWithInactiveTimeout: "+object.Name, func() {}),
	}

	var err error
	srwit.gcsReader, err = bucket.NewReaderWithReadHandle(
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

	srwit.timer = time.AfterFunc(srwit.timeDuration, srwit.closeReaderDueToInactivity)
	return srwit, err
}

func (srwit *storageReaderWithInactiveTimeout) Read(p []byte) (n int, err error) {
	srwit.mu.Lock()
	defer srwit.mu.Unlock()

	srwit.timer.Stop()
	if srwit.gcsReader == nil {
		srwit.gcsReader, err = srwit.bucket.NewReaderWithReadHandle(
			srwit.parentContext,
			&gcs.ReadObjectRequest{
				Name:       srwit.object.Name,
				Generation: srwit.object.Generation,
				Range: &gcs.ByteRange{
					Start: uint64(srwit.start + srwit.seen),
					Limit: uint64(srwit.end),
				},
				ReadCompressed: srwit.object.HasContentEncodingGzip(),
				ReadHandle:     srwit.readHandle,
			})

		if err != nil {
			err = fmt.Errorf("NewReaderWithReadHandle: %w", err)
			return
		}
	}

	n, err = srwit.gcsReader.Read(p)
	if !srwit.timer.Reset(srwit.timeDuration) {
		srwit.discardInactivityCallbackOnce = true
	}
	srwit.seen += int64(n)
	return
}

func (srwit *storageReaderWithInactiveTimeout) Close() (err error) {
	srwit.mu.Lock()
	defer srwit.mu.Unlock()

	srwit.timer.Stop()
	if srwit.gcsReader == nil {
		return
	}

	err = srwit.gcsReader.Close()
	if err != nil {
		return fmt.Errorf("close reader: %w", err)
	}
	return
}

func (srwit *storageReaderWithInactiveTimeout) ReadHandle() (rh storagev2.ReadHandle) {
	srwit.mu.Lock()
	defer srwit.mu.Unlock()

	if srwit.gcsReader == nil {
		return srwit.readHandle
	}

	return srwit.gcsReader.ReadHandle()
}

func (srwit *storageReaderWithInactiveTimeout) closeReaderDueToInactivity() {
	srwit.mu.Lock()
	defer srwit.mu.Unlock()

	// Discard the execution if there are two floating execution of the function.
	if srwit.discardInactivityCallbackOnce {
		srwit.discardInactivityCallbackOnce = false
	}

	if srwit.gcsReader == nil {
		return
	}

	// Update the readHandle before closing reader, keep the behavior similar to random_reader.go
	srwit.readHandle = srwit.gcsReader.ReadHandle()

	logger.Infof("Closing the reader (%s) due to inactivity for %0.1fs.\n", srwit.object.Name, srwit.timeDuration.Seconds())
	err := srwit.gcsReader.Close()
	if err != nil {
		logger.Warnf("error while closing reader: %v", err)
	}
	srwit.gcsReader = nil
}
