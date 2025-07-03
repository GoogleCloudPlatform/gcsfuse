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

package common

import (
	"context"
	"time"
)

func NewNoopMetrics() MetricHandle {
	var n noopMetrics
	return &n
}

type noopMetrics struct{}

func (*noopMetrics) GcsReadBytesCount(_ int64)                                        {}
func (*noopMetrics) GcsReaderCount(_ int64, _ string)                                 {}
func (*noopMetrics) GcsRequestCount(_ int64, _ string)                                {}
func (*noopMetrics) GcsRequestLatencies(_ context.Context, _ time.Duration, _ string) {}
func (*noopMetrics) GcsReadCount(_ int64, _ string)                                   {}
func (*noopMetrics) GcsDownloadBytesCount(_ int64, _ string)                          {}
func (*noopMetrics) GcsRetryCount(_ int64, _ string)                                  {}

func (*noopMetrics) FsOpsCount(_ int64, _ string)                              {}
func (*noopMetrics) FsOpsLatency(_ context.Context, _ time.Duration, _ string) {}
func (*noopMetrics) FsOpsErrorCount(_ int64, _ string, _ string)               {}

func (*noopMetrics) FileCacheReadCount(_ int64, _ string, _ string)                      {}
func (*noopMetrics) FileCacheReadBytesCount(_ int64, _ string)                           {}
func (*noopMetrics) FileCacheReadLatencies(_ context.Context, _ time.Duration, _ string) {}
