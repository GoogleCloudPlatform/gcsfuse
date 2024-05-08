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

	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/monitor/tags"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/storage/gcs"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	// OpenCensus measures
	readBytesCount = stats.Int64("gcs/read_bytes_count", "The number of bytes read from GCS objects.", stats.UnitBytes)
	readerCount    = stats.Int64("gcs/reader_count", "The number of GCS object readers opened or closed.", stats.UnitDimensionless)
	requestCount   = stats.Int64("gcs/request_count", "The number of GCS requests processed.", stats.UnitDimensionless)
	requestLatency = stats.Float64("gcs/request_latency", "The latency of a GCS request.", stats.UnitMilliseconds)
)

// Initialize the metrics.
func init() {
	// OpenCensus views (aggregated measures)
	if err := view.Register(
		&view.View{
			Name:        "gcs/read_bytes_count",
			Measure:     readBytesCount,
			Description: "The cumulative number of bytes read from GCS objects.",
			Aggregation: view.Sum(),
		},
		&view.View{
			Name:        "gcs/reader_count",
			Measure:     readerCount,
			Description: "The cumulative number of GCS object readers opened or closed.",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.IOMethod},
		},
		&view.View{
			Name:        "gcs/request_count",
			Measure:     requestCount,
			Description: "The cumulative number of GCS requests processed.",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.GCSMethod},
		},
		&view.View{
			Name:        "gcs/request_latencies",
			Measure:     requestLatency,
			Description: "The cumulative distribution of the GCS request latencies.",
			Aggregation: ochttp.DefaultLatencyDistribution,
			TagKeys:     []tag.Key{tags.GCSMethod},
		}); err != nil {
		fmt.Printf("Failed to register OpenCensus metrics for GCS client library: %v", err)
	}
}

// recordRequest records a request and its latency.
func recordRequest(ctx context.Context, method string, start time.Time) {
	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.GCSMethod, method),
		},
		requestCount.M(1),
	); err != nil {
		// The error should be caused by a bad tag
		logger.Errorf("Cannot record request count: %v", err)
	}

	latencyUs := time.Since(start).Microseconds()
	latencyMs := float64(latencyUs) / 1000.0
	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.GCSMethod, method),
		},
		requestLatency.M(latencyMs),
	); err != nil {
		// The error should be caused by a bad tag
		logger.Errorf("Cannot record request latency: %v", err)
	}
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

func (mb *monitoringBucket) Type() string {
	return mb.wrapped.Type()
}

func (mb *monitoringBucket) NewReader(
	ctx context.Context,
	req *gcs.ReadObjectRequest) (rc io.ReadCloser, err error) {
	startTime := time.Now()

	rc, err = mb.wrapped.NewReader(ctx, req)
	if err == nil {
		rc = newMonitoringReadCloser(ctx, req.Name, rc)
	}

	recordRequest(ctx, "NewReader", startTime)
	return
}

func (mb *monitoringBucket) CreateObject(
	ctx context.Context,
	req *gcs.CreateObjectRequest) (*gcs.Object, error) {
	startTime := time.Now()
	o, err := mb.wrapped.CreateObject(ctx, req)
	recordRequest(ctx, "CreateObject", startTime)
	return o, err
}

func (mb *monitoringBucket) CopyObject(
	ctx context.Context,
	req *gcs.CopyObjectRequest) (*gcs.Object, error) {
	startTime := time.Now()
	o, err := mb.wrapped.CopyObject(ctx, req)
	recordRequest(ctx, "CopyObject", startTime)
	return o, err
}

func (mb *monitoringBucket) ComposeObjects(
	ctx context.Context,
	req *gcs.ComposeObjectsRequest) (*gcs.Object, error) {
	startTime := time.Now()
	o, err := mb.wrapped.ComposeObjects(ctx, req)
	recordRequest(ctx, "ComposeObjects", startTime)
	return o, err
}

func (mb *monitoringBucket) StatObject(
	ctx context.Context,
	req *gcs.StatObjectRequest) (*gcs.MinObject, *gcs.ExtendedObjectAttributes, error) {
	startTime := time.Now()
	m, e, err := mb.wrapped.StatObject(ctx, req)
	recordRequest(ctx, "StatObject", startTime)
	return m, e, err
}

func (mb *monitoringBucket) ListObjects(
	ctx context.Context,
	req *gcs.ListObjectsRequest) (*gcs.Listing, error) {
	startTime := time.Now()
	listing, err := mb.wrapped.ListObjects(ctx, req)
	recordRequest(ctx, "ListObjects", startTime)
	return listing, err
}

func (mb *monitoringBucket) UpdateObject(
	ctx context.Context,
	req *gcs.UpdateObjectRequest) (*gcs.Object, error) {
	startTime := time.Now()
	o, err := mb.wrapped.UpdateObject(ctx, req)
	recordRequest(ctx, "UpdateObject", startTime)
	return o, err
}

func (mb *monitoringBucket) DeleteObject(
	ctx context.Context,
	req *gcs.DeleteObjectRequest) error {
	startTime := time.Now()
	err := mb.wrapped.DeleteObject(ctx, req)
	recordRequest(ctx, "DeleteObject", startTime)
	return err
}

// recordReader increments the reader count when it's opened or closed.
func recordReader(ctx context.Context, ioMethod string) {
	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.IOMethod, ioMethod),
		},
		readerCount.M(1),
	); err != nil {
		logger.Errorf("Cannot record a reader %v: %v", ioMethod, err)
	}
}

// Monitoring on the object reader
func newMonitoringReadCloser(ctx context.Context, object string, rc io.ReadCloser) io.ReadCloser {
	recordReader(ctx, "opened")
	return &monitoringReadCloser{
		ctx:     ctx,
		object:  object,
		wrapped: rc,
	}
}

type monitoringReadCloser struct {
	ctx     context.Context
	object  string
	wrapped io.ReadCloser
}

func (mrc *monitoringReadCloser) Read(p []byte) (n int, err error) {
	n, err = mrc.wrapped.Read(p)
	if err == nil {
		stats.Record(mrc.ctx, readBytesCount.M(int64(n)))
	}
	return
}

func (mrc *monitoringReadCloser) Close() (err error) {
	err = mrc.wrapped.Close()
	if err != nil {
		return fmt.Errorf("close reader: %w", err)
	}
	recordReader(mrc.ctx, "closed")
	return
}
