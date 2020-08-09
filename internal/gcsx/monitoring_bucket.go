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
)

// Initialize the prometheus metrics.
func init() {
	prometheus.MustRegister(counterGcsRequests)
}

func incrementCounterGcsRequests(bucketName string, method string) {
	counterGcsRequests.With(
		prometheus.Labels{
			"bucket": bucketName,
			"method": method,
		},
	).Inc()
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
	req *gcs.ReadObjectRequest) (io.ReadCloser, error) {
	incrementCounterGcsRequests(mb.Name(), "NewReader")
	return mb.wrapped.NewReader(ctx, req)
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
