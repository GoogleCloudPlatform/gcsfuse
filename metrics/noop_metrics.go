// Copyright 2025 Google LLC
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

// **** DO NOT EDIT - FILE IS AUTO-GENERATED ****
package metrics

import (
	"context"
	"time"
)

type noopMetrics struct{}

func (*noopMetrics) BufferedReadFallbackTriggerCount(inc int64, reason Reason) {}

func (*noopMetrics) BufferedReadReadLatency(ctx context.Context, latency time.Duration) {}

func (*noopMetrics) FileCacheReadBytesCount(inc int64, readType ReadType) {}

func (*noopMetrics) FileCacheReadCount(inc int64, cacheHit bool, readType ReadType) {}

func (*noopMetrics) FileCacheReadLatencies(ctx context.Context, latency time.Duration, cacheHit bool) {
}

func (*noopMetrics) FsOpsCount(inc int64, fsOp FsOp) {}

func (*noopMetrics) FsOpsErrorCount(inc int64, fsErrorCategory FsErrorCategory, fsOp FsOp) {}

func (*noopMetrics) FsOpsLatency(ctx context.Context, latency time.Duration, fsOp FsOp) {}

func (*noopMetrics) GcsDownloadBytesCount(inc int64, readType ReadType) {}

func (*noopMetrics) GcsReadBytesCount(inc int64) {}

func (*noopMetrics) GcsReadCount(inc int64, readType ReadType) {}

func (*noopMetrics) GcsReaderCount(inc int64, ioMethod IoMethod) {}

func (*noopMetrics) GcsRequestCount(inc int64, gcsMethod GcsMethod) {}

func (*noopMetrics) GcsRequestLatencies(ctx context.Context, latency time.Duration, gcsMethod GcsMethod) {
}

func (*noopMetrics) GcsRetryCount(inc int64, retryErrorCategory RetryErrorCategory) {}

func (*noopMetrics) GcsfuseMemoryBytes(inc int64, component Component) {}

func (*noopMetrics) TestUpdownCounter(inc int64) {}

func (*noopMetrics) TestUpdownCounterWithAttrs(inc int64, requestType RequestType) {}

func NewNoopMetrics() MetricHandle {
	var n noopMetrics
	return &n
}
