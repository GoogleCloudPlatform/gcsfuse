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

package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// VerifyCounterMetric finds a counter metric and verifies that the data point
// matching the provided attributes has the expected value.
func VerifyCounterMetric(t *testing.T, ctx context.Context, reader *metric.ManualReader, metricName string, attrs attribute.Set, expectedValue int64) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := reader.Collect(ctx, &rm)
	require.NoError(t, err, "reader.Collect")
	encoder := attribute.DefaultEncoder()
	expectedKey := attrs.Encoded(encoder)

	require.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	require.NotEmpty(t, rm.ScopeMetrics[0].Metrics, "expected at least 1 metric")

	foundMetric := false
	for _, m := range rm.ScopeMetrics[0].Metrics {
		if m.Name == metricName {
			foundMetric = true
			data, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "metric %s is not a Sum[int64], but %T", metricName, m.Data)

			foundDataPoint := false
			for _, dp := range data.DataPoints {
				if dp.Attributes.Encoded(encoder) == expectedKey {
					foundDataPoint = true
					assert.Equal(t, expectedValue, dp.Value, "metric value mismatch for attributes: %s", attrs.Encoded(encoder))
					break
				}
			}

			require.True(t, foundDataPoint, "Data point for attributes %v not found in %s metric", attrs, metricName)
			break
		}
	}

	require.True(t, foundMetric, "metric %s not found", metricName)
}

// VerifyHistogramMetric finds a histogram metric and verifies that the data point
// matching the provided attributes has the expected count.
func VerifyHistogramMetric(t *testing.T, ctx context.Context, reader *metric.ManualReader, metricName string, attrs attribute.Set, expectedCount uint64) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := reader.Collect(ctx, &rm)
	require.NoError(t, err, "reader.Collect")
	encoder := attribute.DefaultEncoder()
	expectedKey := attrs.Encoded(encoder)

	require.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	require.NotEmpty(t, rm.ScopeMetrics[0].Metrics, "expected at least 1 metric")

	foundMetric := false
	for _, m := range rm.ScopeMetrics[0].Metrics {
		if m.Name == metricName {
			foundMetric = true
			data, ok := m.Data.(metricdata.Histogram[int64])
			require.True(t, ok, "metric %s is not a Histogram[int64], but %T", metricName, m.Data)

			foundDataPoint := false
			for _, dp := range data.DataPoints {
				if dp.Attributes.Encoded(encoder) == expectedKey {
					foundDataPoint = true
					assert.Equal(t, expectedCount, dp.Count, "metric count mismatch for attributes: %s", attrs.Encoded(encoder))
					break
				}
			}
			require.True(t, foundDataPoint, "Data point for attributes %v not found in %s metric", attrs, metricName)
			break
		}
	}
	require.True(t, foundMetric, "metric %s not found", metricName)
}
