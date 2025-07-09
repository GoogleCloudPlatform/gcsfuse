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

package common

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func setupOTel(ctx context.Context, t *testing.T) (*otelMetrics, *metric.ManualReader) {
	t.Helper()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)

	m, err := NewOTelMetrics(ctx, 1, 100)
	require.NoError(t, err)
	return m, reader
}

// gatherNonZeroCounterMetrics collects all non-zero counter metrics from the reader.
// It returns a map where the key is the metric name, and the value is another map.
// The inner map's key is a sorted, semicolon-separated string of attributes,
// and the value is the counter's value.
func gatherNonZeroCounterMetrics(ctx context.Context, t *testing.T, rd *metric.ManualReader) map[string]map[string]int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := rd.Collect(ctx, &rm)
	require.NoError(t, err)

	results := make(map[string]map[string]int64)

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			// We are interested in Sum[int64] which corresponds to int_counter.
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}

			metricMap := make(map[string]int64)
			for _, dp := range sum.DataPoints {
				if dp.Value == 0 {
					continue
				}

				var parts []string
				for _, kv := range dp.Attributes.ToSlice() {
					parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
				}
				sort.Strings(parts)
				key := strings.Join(parts, ";")

				metricMap[key] = dp.Value
			}

			if len(metricMap) > 0 {
				results[m.Name] = metricMap
			}
		}
	}

	return results
}

func TestFsOpsCount(t *testing.T) {
	ctx := context.Background()
	m, rd := setupOTel(ctx, t)

	m.FsOpsCount(5, "BatchForget")
	m.FsOpsCount(2, "CreateFile")
	m.FsOpsCount(3, "BatchForget")

	m.Flush(ctx)

	// Assert
	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	opsCount, ok := metrics["fs/ops_count"]
	assert.True(t, ok, "fs/ops_count metric not found")
	expected := map[string]int64{
		"fs_op=BatchForget": 8,
		"fs_op=CreateFile":  2,
	}
	assert.Equal(t, expected, opsCount)

}
