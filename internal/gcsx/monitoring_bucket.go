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

package gcsx

import (
	"context"
	"io"
	"time"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	counterGcsRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcsfuse_gcs_requests",
			Help: "Number of GCS requests.",
		},
		[]string{ // labels
			"bucket",
			"method",
		},
	)
	counterBytesRead = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcsfuse_bytes_read",
			Help: "Number of bytes read from GCS.",
		},
		[]string{ // labels
			"bucket",
			"object",
		},
	)
	counterObjectReadersCreated = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gcsfuse_object_readers_created",
			Help: "Number of object readers created.",
		},
	)
	counterObjectReadersClosed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gcsfuse_object_readers_closed",
			Help: "Number of object readers already closed.",
		},
	)
	latencyNewReader = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "gcsfuse_object_new_reader_latency",
			Help: "The latency of creating a GCS object reader in ms.",

			// 32 buckets: [0.1ms, 0.15ms, ..., 28.8s, +Inf]
			Buckets: prometheus.ExponentialBuckets(0.1, 1.5, 32),
		},
	)
	latencyRead = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "gcsfuse_object_read_latency",
			Help: "The latency of reading once by the reader in ms.",

			// 32 buckets: [0.1ms, 0.15ms, ..., 28.8s, +Inf]
			Buckets: prometheus.ExponentialBuckets(0.1, 1.5, 32),
		},
	)
)

// Initialize the prometheus metrics.
func init() {
	prometheus.MustRegister(counterGcsRequests)
	prometheus.MustRegister(counterBytesRead)
	prometheus.MustRegister(counterObjectReadersCreated)
	prometheus.MustRegister(counterObjectReadersClosed)
	prometheus.MustRegister(latencyNewReader)
	prometheus.MustRegister(latencyRead)
}

func incrementCounterGcsRequests(bucketName string, method string) {
	counterGcsRequests.With(
		prometheus.Labels{
			"bucket": bucketName,
			"method": method,
		},
	).Inc()
}

func incrementCounterBytesRead(bucketName string, object string, bytes int) {
	counterBytesRead.With(
		prometheus.Labels{
			"bucket": bucketName,
			"object": object,
		},
	).Add(float64(bytes))
}

func recordLatency(metric prometheus.Histogram, start time.Time) {
	latency := float64(time.Since(start).Milliseconds())
	metric.Observe(latency)
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
	defer recordLatency(latencyNewReader, time.Now())

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
	counterObjectReadersCreated.Inc()
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
	defer recordLatency(latencyRead, time.Now())

	n, err = mrc.wrapped.Read(p)
	if err == nil {
		incrementCounterBytesRead(mrc.bucketName, mrc.object, n)
	}
	return
}

func (mrc *monitoringReadCloser) Close() (err error) {
	err = mrc.wrapped.Close()
	if err == nil {
		counterObjectReadersClosed.Inc()
	}
	return
}
