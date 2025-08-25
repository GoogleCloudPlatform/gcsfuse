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

func (*noopMetrics) BufferedReadDownloadBlockLatency(ctx context.Context, duration time.Duration, status string) {
}

func (*noopMetrics) BufferedReadFallbackTriggerCount(inc int64, reason string) {}

func (*noopMetrics) BufferedReadReadLatency(ctx context.Context, duration time.Duration) {}

func (*noopMetrics) BufferedReadScheduledBlockCount(inc int64, status string) {}

func (*noopMetrics) FileCacheReadBytesCount(inc int64, readType string) {}

func (*noopMetrics) FileCacheReadCount(inc int64, cacheHit bool, readType string) {}

func (*noopMetrics) FileCacheReadLatencies(ctx context.Context, duration time.Duration, cacheHit bool) {
}

func (*noopMetrics) FsOpsCount(inc int64, fsOp string) {}

func (*noopMetrics) FsOpsErrorCount(inc int64, fsErrorCategory string, fsOp string) {}

func (*noopMetrics) FsOpsLatency(ctx context.Context, duration time.Duration, fsOp string) {}

func (*noopMetrics) GcsDownloadBytesCount(inc int64, readType string) {}

func (*noopMetrics) GcsReadBytesCount(inc int64) {}

func (*noopMetrics) GcsReadCount(inc int64, readType string) {}

func (*noopMetrics) GcsReaderCount(inc int64, ioMethod string) {}

func (*noopMetrics) GcsRequestCount(inc int64, gcsMethod string) {}

func (*noopMetrics) GcsRequestLatencies(ctx context.Context, duration time.Duration, gcsMethod string) {
}

func (*noopMetrics) GcsRetryCount(inc int64, retryErrorCategory string) {}

func NewNoopMetrics() MetricHandle {
	var n noopMetrics
	return &n
}
