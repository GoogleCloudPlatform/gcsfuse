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

package monitor

import (
	"context"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/monitor/tags"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	// OpenCensus measures
	requestCountOC   = stats.Int64("gcs/request_count", "The number of GCS requests processed.", stats.UnitDimensionless)
	requestLatencyOC = stats.Float64("gcs/request_latency", "The latency of a GCS request.", stats.UnitMilliseconds)
)

// Initialize the metrics.
func init() {
	// OpenCensus views (aggregated measures)
	if err := view.Register(
		&view.View{
			Name:        "gcs/request_count",
			Measure:     requestCountOC,
			Description: "The cumulative number of GCS requests processed.",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tags.GCSMethod},
		},
		&view.View{
			Name:        "gcs/request_latencies",
			Measure:     requestLatencyOC,
			Description: "The cumulative distribution of the GCS request latencies.",
			Aggregation: ochttp.DefaultLatencyDistribution,
			TagKeys:     []tag.Key{tags.GCSMethod},
		}); err != nil {
		logger.Errorf("Failed to register OpenCensus metrics for GCS client library: %v", err)
	}
}

// recordRequest records a request and its latency.
func recordRequest(ctx context.Context, method string, start time.Time) {
	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.GCSMethod, method),
		},
		requestCountOC.M(1),
	); err != nil {
		// The error should be caused by a bad tag
		logger.Errorf("Cannot record request count: %v", err)
	}

	latencyUs := time.Since(start).Microseconds()
	latencyMs := float64(latencyUs) / 1000.0
	if err := stats.RecordWithTags(
		ctx,
		[]tag.Mutator{
			tag.Upsert(tags.GCSMethod, method),
		},
		requestLatencyOC.M(latencyMs),
	); err != nil {
		// The error should be caused by a bad tag
		logger.Errorf("Cannot record request latency: %v", err)
	}
}
