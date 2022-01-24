// Copyright 2020 Google Inc. All Rights Reserved.
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
	"io"
	"time"

	"github.com/jacobsa/gcloud/gcs"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	methodName = tag.MustNewKey("method_name")
	bucketName = tag.MustNewKey("bucket_name")
)

var (
	requestCount = stats.Int64(
		"gcs_requests",
		"Number of GCS requests.",
		stats.UnitDimensionless)
	bytesRead = stats.Int64(
		"gcs_bytes_read",
		"Number of bytes read from GCS.",
		stats.UnitBytes)
	objectReadersCreated = stats.Int64(
		"object_readers_created",
		"Number of object readers created.",
		stats.UnitDimensionless)
	objectReadersClosed = stats.Int64(
		"object_readers_closed",
		"Number of object readers closed.",
		stats.UnitDimensionless)
	latency = stats.Float64(
		"gcs_latency",
		"The latency of the method being executed in ms.",
		stats.UnitMilliseconds)
)

// Initialize the metrics.
func init() {
	if err := view.Register(
		&view.View{
			Name:        requestCount.Name(),
			Measure:     requestCount,
			Description: requestCount.Description(),
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{methodName, bucketName},
		},
		&view.View{
			Name:        bytesRead.Name(),
			Measure:     bytesRead,
			Description: bytesRead.Description(),
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{bucketName},
		},
		&view.View{
			Name:        objectReadersCreated.Name(),
			Measure:     objectReadersCreated,
			Description: objectReadersCreated.Description(),
			Aggregation: view.Sum(),
		},
		&view.View{
			Name:        objectReadersClosed.Name(),
			Measure:     objectReadersClosed,
			Description: objectReadersClosed.Description(),
			Aggregation: view.Sum(),
		},
		&view.View{
			Name:        latency.Name(),
			Measure:     latency,
			Description: latency.Description(),
			Aggregation: view.Distribution(),
			TagKeys:     []tag.Key{methodName, bucketName},
		}); err != nil {
		fmt.Printf("Failed to register metrics in the monitoring bucket\n")
	}
}

func incrementCounterGcsRequests(bucket string, method string) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{
			tag.Upsert(methodName, method),
			tag.Upsert(bucketName, bucket),
		},
		requestCount.M(1),
	)
}

func incrementCounterBytesRead(bucket string, bytes int) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{tag.Upsert(bucketName, bucket)},
		bytesRead.M(int64(bytes)),
	)
}

func recordLatency(method string, start time.Time) {
	stats.RecordWithTags(
		context.Background(),
		[]tag.Mutator{tag.Upsert(methodName, method)},
		latency.M(float64(time.Since(start).Milliseconds())),
	)
}

// NewMonitoringBucket returns a gcs.Bucket that exports metrics for monitoring
func NewMonitoringBucket(b gcs.Bucket) gcs.Bucket {
	return &monitoringBucket{
		wrapped: b,
	}
}

type monitoringBucket struct {
	wrapped gcs.Bucket
}

func (mb *monitoringBucket) Name() string {
	return mb.wrapped.Name()
}

func (mb *monitoringBucket) NewReader(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (rc io.ReadCloser, err error) {
	incrementCounterGcsRequests(mb.Name(), "NewReader")
	defer recordLatency("NewReader", time.Now())

	rc, err = mb.wrapped.NewReader(ctx, req)
	if err == nil {
		rc = newMonitoringReadCloser(mb.Name(), req.Name, rc)
	}
	return
}

func (mb *monitoringBucket) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (*gcs.Object, error) {
	incrementCounterGcsRequests(mb.Name(), "CreateObject")
	return mb.wrapped.CreateObject(ctx, req)
}

func (mb *monitoringBucket) CopyObject(
	ctx context.Context,
	req *gcs.CopyObjectRequest) (*gcs.Object, error) {
	incrementCounterGcsRequests(mb.Name(), "CopyObject")
	return mb.wrapped.CopyObject(ctx, req)
}

func (mb *monitoringBucket) ComposeObjects(
	ctx context.Context,
	req *gcs.ComposeObjectsRequest) (*gcs.Object, error) {
	incrementCounterGcsRequests(mb.Name(), "ComposeObjects")
	return mb.wrapped.ComposeObjects(ctx, req)
}

func (mb *monitoringBucket) StatObject(
	ctx context.Context,
	req *gcs.StatObjectRequest) (*gcs.Object, error) {
	incrementCounterGcsRequests(mb.Name(), "StatObject")
	return mb.wrapped.StatObject(ctx, req)
}

func (mb *monitoringBucket) ListObjects(
	ctx context.Context,
	req *gcs.ListObjectsRequest) (*gcs.Listing, error) {
	incrementCounterGcsRequests(mb.Name(), "ListObjects")
	return mb.wrapped.ListObjects(ctx, req)
}

func (mb *monitoringBucket) UpdateObject(
	ctx context.Context,
	req *gcs.UpdateObjectRequest) (*gcs.Object, error) {
	incrementCounterGcsRequests(mb.Name(), "UpdateObject")
	return mb.wrapped.UpdateObject(ctx, req)
}

func (mb *monitoringBucket) DeleteObject(
	ctx context.Context,
	req *gcs.DeleteObjectRequest) error {
	incrementCounterGcsRequests(mb.Name(), "DeleteObject")
	return mb.wrapped.DeleteObject(ctx, req)
}

func newMonitoringReadCloser(
	bucketName string,
	object string,
	rc io.ReadCloser) io.ReadCloser {
	stats.Record(context.Background(), objectReadersCreated.M(1))
	return &monitoringReadCloser{
		bucketName: bucketName,
		object:     object,
		wrapped:    rc,
	}
}

type monitoringReadCloser struct {
	bucketName string
	object     string
	wrapped    io.ReadCloser
}

func (mrc *monitoringReadCloser) Read(p []byte) (n int, err error) {
	defer recordLatency("Read", time.Now())

	n, err = mrc.wrapped.Read(p)
	if err == nil {
		incrementCounterBytesRead(mrc.bucketName, n)
	}
	return
}

func (mrc *monitoringReadCloser) Close() (err error) {
	err = mrc.wrapped.Close()
	if err == nil {
		stats.Record(context.Background(), objectReadersClosed.M(1))
	}
	return
}
