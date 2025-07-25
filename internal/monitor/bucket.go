// Copyright 2020 Google LLC
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

package monitor

import (
	"context"
	"fmt"
	"time"

	storagev2 "cloud.google.com/go/storage"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// recordRequest records a request and its latency.
func recordRequest(ctx context.Context, metricHandle metrics.MetricHandle, method string, start time.Time) {
	metricHandle.GCSRequestCount(ctx, 1, method)

	metricHandle.GCSRequestLatency(ctx, time.Since(start), method)
}

func CaptureMultiRangeDownloaderMetrics(ctx context.Context, metricHandle metrics.MetricHandle, method string, start time.Time) {
	recordRequest(ctx, metricHandle, method, start)
}

// NewMonitoringBucket returns a gcs.Bucket that exports metrics for monitoring
func NewMonitoringBucket(b gcs.Bucket, m metrics.MetricHandle) gcs.Bucket {
	return &monitoringBucket{
		wrapped:      b,
		metricHandle: m,
	}
}

type monitoringBucket struct {
	wrapped      gcs.Bucket
	metricHandle metrics.MetricHandle
}

func (mb *monitoringBucket) Name() string {
	return mb.wrapped.Name()
}

func (mb *monitoringBucket) BucketType() gcs.BucketType {
	return mb.wrapped.BucketType()
}

func setupReader(ctx context.Context, mb *monitoringBucket, req *gcs.ReadObjectRequest, method string) (gcs.StorageReader, error) {
	startTime := time.Now()

	rc, err := mb.wrapped.NewReaderWithReadHandle(ctx, req)

	if err == nil {
		rc = newMonitoringReadCloser(ctx, req.Name, rc, mb.metricHandle)
	}

	recordRequest(ctx, mb.metricHandle, method, startTime)
	return rc, err
}

func (mb *monitoringBucket) NewReaderWithReadHandle(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (rd gcs.StorageReader, err error) {
	// Using NewReader here also as NewReader() method is not used and will be removed.
	rd, err = setupReader(ctx, mb, req, "NewReader")
	return
}

func (mb *monitoringBucket) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (*gcs.Object, error) {
	startTime := time.Now()
	o, err := mb.wrapped.CreateObject(ctx, req)
	recordRequest(ctx, mb.metricHandle, "CreateObject", startTime)
	return o, err
}

func (mb *monitoringBucket) CreateObjectChunkWriter(ctx context.Context, req *gcs.CreateObjectRequest, chunkSize int, callBack func(bytesUploadedSoFar int64)) (gcs.Writer, error) {
	startTime := time.Now()
	wc, err := mb.wrapped.CreateObjectChunkWriter(ctx, req, chunkSize, callBack)
	recordRequest(ctx, mb.metricHandle, "CreateObjectChunkWriter", startTime)
	return wc, err
}

func (mb *monitoringBucket) CreateAppendableObjectWriter(ctx context.Context, req *gcs.CreateObjectChunkWriterRequest) (gcs.Writer, error) {
	startTime := time.Now()
	wc, err := mb.wrapped.CreateAppendableObjectWriter(ctx, req)
	recordRequest(ctx, mb.metricHandle, "CreateAppendableObjectWriter", startTime)
	return wc, err
}

func (mb *monitoringBucket) FinalizeUpload(ctx context.Context, w gcs.Writer) (*gcs.MinObject, error) {
	startTime := time.Now()
	o, err := mb.wrapped.FinalizeUpload(ctx, w)
	recordRequest(ctx, mb.metricHandle, "FinalizeUpload", startTime)
	return o, err
}

func (mb *monitoringBucket) FlushPendingWrites(ctx context.Context, w gcs.Writer) (*gcs.MinObject, error) {
	startTime := time.Now()
	o, err := mb.wrapped.FlushPendingWrites(ctx, w)
	recordRequest(ctx, mb.metricHandle, "FlushPendingWrites", startTime)
	return o, err
}

func (mb *monitoringBucket) CopyObject(
	ctx context.Context,
	req *gcs.CopyObjectRequest) (*gcs.Object, error) {
	startTime := time.Now()
	o, err := mb.wrapped.CopyObject(ctx, req)
	recordRequest(ctx, mb.metricHandle, "CopyObject", startTime)
	return o, err
}

func (mb *monitoringBucket) ComposeObjects(
	ctx context.Context,
	req *gcs.ComposeObjectsRequest) (*gcs.Object, error) {
	startTime := time.Now()
	o, err := mb.wrapped.ComposeObjects(ctx, req)
	recordRequest(ctx, mb.metricHandle, "ComposeObjects", startTime)
	return o, err
}

func (mb *monitoringBucket) StatObject(
	ctx context.Context,
	req *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {
	startTime := time.Now()
	m, e, err := mb.wrapped.StatObject(ctx, req)
	recordRequest(ctx, mb.metricHandle, "StatObject", startTime)
	return m, e, err
}

func (mb *monitoringBucket) ListObjects(
	ctx context.Context,
	req *gcs.ListObjectsRequest) (*gcs.Listing, error) {
	startTime := time.Now()
	listing, err := mb.wrapped.ListObjects(ctx, req)
	recordRequest(ctx, mb.metricHandle, "ListObjects", startTime)
	return listing, err
}

func (mb *monitoringBucket) UpdateObject(
	ctx context.Context,
	req *gcs.UpdateObjectRequest) (*gcs.Object, error) {
	startTime := time.Now()
	o, err := mb.wrapped.UpdateObject(ctx, req)
	recordRequest(ctx, mb.metricHandle, "UpdateObject", startTime)
	return o, err
}

func (mb *monitoringBucket) DeleteObject(
	ctx context.Context,
	req *gcs.DeleteObjectRequest) error {
	startTime := time.Now()
	err := mb.wrapped.DeleteObject(ctx, req)
	recordRequest(ctx, mb.metricHandle, "DeleteObject", startTime)
	return err
}

func (mb *monitoringBucket) MoveObject(ctx context.Context, req *gcs.MoveObjectRequest) (*gcs.Object, error) {
	startTime := time.Now()
	o, err := mb.wrapped.MoveObject(ctx, req)
	recordRequest(ctx, mb.metricHandle, "MoveObject", startTime)
	return o, err
}

func (mb *monitoringBucket) DeleteFolder(ctx context.Context, folderName string) error {
	startTime := time.Now()
	err := mb.wrapped.DeleteFolder(ctx, folderName)
	recordRequest(ctx, mb.metricHandle, "DeleteFolder", startTime)
	return err
}

func (mb *monitoringBucket) GetFolder(ctx context.Context, folderName string) (*gcs.Folder, error) {
	startTime := time.Now()
	folder, err := mb.wrapped.GetFolder(ctx, folderName)
	recordRequest(ctx, mb.metricHandle, "GetFolder", startTime)
	return folder, err
}

func (mb *monitoringBucket) CreateFolder(ctx context.Context, folderName string) (*gcs.Folder, error) {
	startTime := time.Now()
	folder, err := mb.wrapped.CreateFolder(ctx, folderName)
	recordRequest(ctx, mb.metricHandle, "CreateFolder", startTime)
	return folder, err
}

func (mb *monitoringBucket) RenameFolder(ctx context.Context, folderName string, destinationFolderId string) (o *gcs.Folder, err error) {
	startTime := time.Now()
	o, err = mb.wrapped.RenameFolder(ctx, folderName, destinationFolderId)
	recordRequest(ctx, mb.metricHandle, "RenameFolder", startTime)
	return
}

func (mb *monitoringBucket) NewMultiRangeDownloader(
	ctx context.Context, req *gcs.MultiRangeDownloaderRequest) (mrd gcs.MultiRangeDownloader, err error) {
	startTime := time.Now()
	mrd, err = mb.wrapped.NewMultiRangeDownloader(ctx, req)
	recordRequest(ctx, mb.metricHandle, "NewMultiRangeDownloader", startTime)
	return
}

func (mb *monitoringBucket) GCSName(obj *gcs.MinObject) string {
	return mb.wrapped.GCSName(obj)
}

// recordReader increments the reader count when it's opened or closed.
func recordReader(ctx context.Context, metricHandle metrics.MetricHandle, ioMethod string) {
	metricHandle.GCSReaderCount(ctx, 1, ioMethod)
}

// Monitoring on the object reader
func newMonitoringReadCloser(ctx context.Context, object string, rc gcs.StorageReader, metricHandle metrics.MetricHandle) gcs.StorageReader {
	recordReader(ctx, metricHandle, "opened")
	return &monitoringReadCloser{
		ctx:          ctx,
		object:       object,
		wrapped:      rc,
		metricHandle: metricHandle,
	}
}

type monitoringReadCloser struct {
	ctx          context.Context
	object       string
	wrapped      gcs.StorageReader
	metricHandle metrics.MetricHandle
}

func (mrc *monitoringReadCloser) Read(p []byte) (n int, err error) {
	n, err = mrc.wrapped.Read(p)
	mrc.metricHandle.GCSReadBytesCount(mrc.ctx, int64(n))
	return
}

func (mrc *monitoringReadCloser) Close() (err error) {
	err = mrc.wrapped.Close()
	if err != nil {
		return fmt.Errorf("close reader: %w", err)
	}
	recordReader(mrc.ctx, mrc.metricHandle, "closed")
	return
}

func (mrc *monitoringReadCloser) ReadHandle() (rh storagev2.ReadHandle) {
	rh = mrc.wrapped.ReadHandle()
	recordReader(mrc.ctx, mrc.metricHandle, "ReadHandle")
	return
}
