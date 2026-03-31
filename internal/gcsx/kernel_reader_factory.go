// Copyright 2026 Google LLC
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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// NewKernelReader creates a new kernel-optimized reader based on the bucket type.
// For Zonal (Rapid) buckets, it uses the MRD-based kernel reader to maximize
// throughput via connection pooling and multi-range downloads.
// For Standard (Regional) buckets, it uses the Range-based kernel reader which
// leverages the kernel read-ahead and creates specific range requests.
func NewKernelReader(bucket gcs.Bucket, object *gcs.MinObject, mrdInstance *MrdInstance, metricsHandle metrics.MetricHandle) Reader {
	if bucket.BucketType().Zonal {
		return NewMrdKernelReader(mrdInstance, metricsHandle)
	}
	return NewRangeKernelReader(bucket, object, metricsHandle)
}
