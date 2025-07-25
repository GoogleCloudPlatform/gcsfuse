// Copyright 2024 Google LLC
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

package metrics

import (
	"context"
	"time"
)

func NewNoopMetrics() MetricHandle {
	var n noopMetrics
	return &n
}

type noopMetrics struct{}

func (*noopMetrics) GCSReadBytesCount(_ context.Context, _ int64)                   {}
func (*noopMetrics) GCSReaderCount(_ context.Context, _ int64, _ string)            {}
func (*noopMetrics) GCSRequestCount(_ context.Context, _ int64, _ string)           {}
func (*noopMetrics) GCSRequestLatency(_ context.Context, _ time.Duration, _ string) {}
func (*noopMetrics) GCSReadCount(_ context.Context, _ int64, _ string)              {}
func (*noopMetrics) GCSDownloadBytesCount(_ context.Context, _ int64, _ string)     {}
func (*noopMetrics) GCSRetryCount(_ context.Context, _ int64, _ string)             {}

func (*noopMetrics) OpsCount(_ context.Context, _ int64, _ string)                  {}
func (*noopMetrics) OpsLatency(_ context.Context, _ time.Duration, _ string)        {}
func (*noopMetrics) OpsErrorCount(_ context.Context, _ int64, _ FSOpsErrorCategory) {}

func (*noopMetrics) FileCacheReadCount(_ context.Context, _ int64, attrs CacheHitReadType) {}
func (*noopMetrics) FileCacheReadBytesCount(_ context.Context, _ int64, _ string)          {}
func (*noopMetrics) FileCacheReadLatency(_ context.Context, _ time.Duration, _ bool)       {}
