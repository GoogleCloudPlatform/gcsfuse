package gcsx

import (
	"fmt"
	"time"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"golang.org/x/net/context"
)

/**
Corner case:
	1. closeReaderDueToInactivity call happens because of inactivity.
	2. In the meantime, new Read() call comes.
	3. Two options:
		(a) Read() happens before the closeReaderDueToInactivity: data will be served from the previous reader, and then later closed.
		(b) Read() happens after the closeReaderDueToInactivity: first active reader will be closed and then read will create a new reader.


	In both cases, there will be only one close, but shouldn't be a problem as it only can happen in the rare scenario.
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

	mu locker.Locker
}

func NewStorageReaderWithInactiveTimeout(ctx context.Context, bucket gcs.Bucket, object *gcs.MinObject, readHandle []byte, start int64, end int64, timeout time.Duration) (gcs.StorageReader, error) {
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
	srwit.timer.Reset(srwit.timeDuration)
	srwit.seen += int64(n)
	return
}

func (srwit *storageReaderWithInactiveTimeout) Close() (err error) {
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
	if srwit.gcsReader == nil {
		return srwit.readHandle
	}

	return srwit.gcsReader.ReadHandle()
}

func (srwit *storageReaderWithInactiveTimeout) closeReaderDueToInactivity() {
	srwit.mu.Lock()
	defer srwit.mu.Unlock()

	if srwit.gcsReader == nil {
		return
	}
	logger.Infof("Closing the reader (%s) due to inactivity for %0.1fs.\n", srwit.object.Name, srwit.timeDuration.Seconds())
	err := srwit.gcsReader.Close()
	if err != nil {
		logger.Warnf("error while closing reader: %v", err)
	}
	srwit.gcsReader = nil
}
