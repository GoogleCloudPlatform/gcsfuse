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

package inode

import (
	"math"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/metrics"
	"github.com/googlecloudplatform/gcsfuse/v3/tracing"
	"golang.org/x/sync/semaphore"
)

func getWriteContext() *WriteContext {
	return &WriteContext{
		Config:             &cfg.Config{},
		GlobalMaxBlocksSem: semaphore.NewWeighted(math.MaxInt64),
		TraceHandle:        tracing.NewNoopTracer(),
		MetricHandle:       metrics.NewNoopMetrics(),
	}
}
