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

package kernel_readers

import (
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/gcsx"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/storage/gcs"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
)

// NewKernelReader creates a new kernel-optimized reader based on the bucket type and protocol.
// For Zonal (Rapid) buckets or Regional buckets with gRPC protocol, it uses the MRD-based kernel reader
// to maximize throughput via connection pooling and multi-range downloads.
// For Standard (Regional) buckets with HTTP protocols, it uses the Range-based kernel reader which
// leverages the kernel read-ahead and creates specific range requests.
func NewKernelReader(
	bucket gcs.Bucket,
	kernelRangeReaderInstance *KernelRangeReaderInstance,
	mrdInstance *gcsx.MrdInstance,
	metricsHandle metrics.MetricHandle,
	protocol cfg.Protocol,
) gcsx.Reader {
	if (bucket != nil && bucket.BucketType().IsRapid()) || protocol == cfg.GRPC {
		return NewKernelMRDReader(mrdInstance, metricsHandle)
	}
	return NewKernelRangeReader(bucket, kernelRangeReaderInstance, metricsHandle)
}

