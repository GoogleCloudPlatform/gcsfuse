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

import "context"

func NewNoopMetrics() MetricHandle {
	var n noopMetrics
	return &n
}

type noopMetrics struct{}

func (n *noopMetrics) GCSReadBytesCount(_ context.Context, _ int64, _ []Attr)       {}
func (n *noopMetrics) GCSReaderCount(_ context.Context, _ int64, _ []Attr)          {}
func (n *noopMetrics) GCSRequestCount(_ context.Context, _ int64, _ []Attr)         {}
func (n *noopMetrics) GCSRequestLatency(_ context.Context, value float64, _ []Attr) {}
func (n *noopMetrics) GCSReadCount(_ context.Context, _ int64, _ []Attr)            {}
func (n *noopMetrics) GCSDownloadBytesCount(_ context.Context, _ int64, _ []Attr)   {}

func (n *noopMetrics) OpsCount(_ context.Context, _ int64, _ []Attr)         {}
func (n *noopMetrics) OpsLatency(_ context.Context, value float64, _ []Attr) {}
func (n *noopMetrics) OpsErrorCount(_ context.Context, _ int64, _ []Attr)    {}

func (n *noopMetrics) FileCacheReadCount(_ context.Context, _ int64, _ []Attr)         {}
func (n *noopMetrics) FileCacheReadBytesCount(_ context.Context, _ int64, _ []Attr)    {}
func (n *noopMetrics) FileCacheReadLatency(_ context.Context, value float64, _ []Attr) {}
