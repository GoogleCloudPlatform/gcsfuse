package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/storageutil"
	"google.golang.org/api/googleapi"
)

type Status int

const (
	Uninitialized Status = iota
	Initialized
	Waiting
	Destroyed
	DefaultChunkSize int = googleapi.DefaultUploadChunkSize
)

type StreamedObjectWriter struct {
	// Implements the following interfaces.
	gcs.ObjectWriter

	status                         Status
	obj                            *storage.ObjectHandle
	req                            *gcs.CreateObjectRequest
	wc                             *storage.Writer
	totalWritInitiatedSoFar        int64
	totalSizeWrittenSucceededSoFar int64
	progressFunc                   func(int64)
}

func NewUninitializedStreamedObjectWriter() StreamedObjectWriter {
	sow := StreamedObjectWriter{}
	sow.status = Uninitialized
	return sow
}

func NewStreamedObjectWriter(ctx context.Context, obj *storage.ObjectHandle, req *gcs.CreateObjectRequest, progressFunc func(int64)) (*StreamedObjectWriter, error) {
	if obj == nil || req == nil {
		return nil, fmt.Errorf("nil input")
	}

	if obj.ObjectName() != req.Name {
		return nil, fmt.Errorf("names of passed object-handle (%s) and CreateObjectRequest (%s) don't match", obj.ObjectName(), req.Name)
	}

	if req.GenerationPrecondition != nil && *req.GenerationPrecondition != 0 {
		return nil, fmt.Errorf("request received for pre-existing object %s, supported only for new objects", req.Name)
	}

	sow := NewUninitializedStreamedObjectWriter()

	sow.obj = obj
	sow.req = req
	sow.progressFunc = progressFunc

	// Create a NewWriter with requested attributes, using Go Storage Client.
	// Chuck size for resumable upload is default i.e. 16MB.
	wc := obj.NewWriter(ctx)
	wc = storageutil.SetAttrsInWriter(wc, req)

	wc.ChunkSize = DefaultChunkSize
	// googleapi.MinUploadChunkSize
	wc.ProgressFunc = func(n int64) {
		sow.totalSizeWrittenSucceededSoFar = n
		logger.Debugf("%d bytes copied so far for object/file %s. chunk-size = %d", n, req.Name, wc.ChunkSize)

		if sow.progressFunc != nil {
			sow.progressFunc(n)
		}
	}

	sow.status = Initialized

	return &sow, nil
}

func (sow StreamedObjectWriter) Status() Status {
	return sow.status
}

func (sow *StreamedObjectWriter) Req() *gcs.CreateObjectRequest {
	return sow.req
}

func (sow StreamedObjectWriter) TotalSizeUploaded() int64 {
	return sow.totalWritInitiatedSoFar
}

func (sow StreamedObjectWriter) BytesWrittenSoFar() int64 {
	return sow.totalSizeWrittenSucceededSoFar
}

// WriteAchunk uploads a given amount of object content
// to given GCS object with a given chunk-size, number of bytes,
// an object writer, and a content-reader.
// It is assumed that all the relevant attributes of the object
// writer have been already by a previous calls to CreateObject
// which created the object-writer.
func (sow *StreamedObjectWriter) Write(contents io.Reader) error {
	if sow.status != Waiting && sow.status != Initialized {
		return fmt.Errorf("improper state for writing to object %s", sow.req.Name)
	}

	bytesToBeWritten := int64(sow.wc.ChunkSize)
	n, err := io.CopyN(sow.wc, contents, bytesToBeWritten)
	sow.totalWritInitiatedSoFar += n
	if err != nil {
		return fmt.Errorf("totalSizeUploaded-so-far=%d, successfully-uploaded-in-last-upload=%d, chunk-size=%d for object %s: %v", sow.totalWritInitiatedSoFar, n, sow.wc.ChunkSize, sow.req.Name, err)
	}

	sow.status = Waiting
	return nil
}

func (sow *StreamedObjectWriter) Close() (o *gcs.Object, err error) {
	if sow.status != Waiting {
		return nil, fmt.Errorf("improper state for finalizing object %s", sow.req.Name)
	}

	// We can't use defer to close the writer, because we need to close the
	// writer successfully before calling Attrs() method of writer.
	if err = sow.wc.Close(); err != nil {
		var gErr *googleapi.Error
		if errors.As(err, &gErr) {
			if gErr.Code == http.StatusPreconditionFailed {
				err = &gcs.PreconditionError{Err: err}
				return
			}
		}
		err = fmt.Errorf("error in closing writer : %w", err)
		return
	}

	sow.status = Destroyed

	attrs := sow.wc.Attrs() // Retrieving the attributes of the created object.
	// Converting attrs to type *Object.
	o = storageutil.ObjectAttrsToBucketObject(attrs)
	return
}
