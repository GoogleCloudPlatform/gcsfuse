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

type verifyConfig struct {
	atLeast bool
	subset  bool
}

// VerifyOption defines functional options for metric verification.
type VerifyOption func(*verifyConfig)

// AtLeast changes the verification to check if the metric value is greater or equal
// to the expected value, rather than exactly equal.
func AtLeast() VerifyOption {
	return func(c *verifyConfig) { c.atLeast = true }
}

// Subset changes the verification to check if the provided attributes are a subset
// of the recorded attributes, rather than an exact match.
func Subset() VerifyOption {
	return func(c *verifyConfig) { c.subset = true }
}

func matchesAttributes(dpAttrs attribute.Set, targetAttrs attribute.Set, subset bool, encoder attribute.Encoder) bool {
	if !subset {
		return dpAttrs.Encoded(encoder) == targetAttrs.Encoded(encoder)
	}
	// Subset matching
	for _, targetKV := range targetAttrs.ToSlice() {
		val, ok := dpAttrs.Value(targetKV.Key)
		if !ok || val.Emit() != targetKV.Value.Emit() {
			return false
		}
	}
	return true
}

func verifyValue[T int64 | uint64](t *testing.T, actual T, expected T, atLeast bool, metricName string, attrs attribute.Set) {
	if atLeast {
		assert.GreaterOrEqual(t, actual, expected, "metric value too low for %s with attributes %v", metricName, attrs)
	} else {
		assert.Equal(t, expected, actual, "metric value mismatch for %s with attributes %v", metricName, attrs)
	}
}

// VerifyCounterMetric finds a counter metric across all scopes and verifies its value.
// By default, it requires an exact attribute match and an exact value match.
// Use AtLeast() or Subset() options to relax these requirements.
func VerifyCounterMetric(t *testing.T, ctx context.Context, reader *metric.ManualReader, metricName string, attrs attribute.Set, expectedValue int64, options ...VerifyOption) {
	t.Helper()
	cfg := &verifyConfig{}
	for _, opt := range options {
		opt(cfg)
	}

	var rm metricdata.ResourceMetrics
	err := reader.Collect(ctx, &rm)
	require.NoError(t, err, "reader.Collect")
	encoder := attribute.DefaultEncoder()

	foundMetric := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricName {
				foundMetric = true
				data, ok := m.Data.(metricdata.Sum[int64])
				require.True(t, ok, "metric %s is not a Sum[int64], but %T", metricName, m.Data)

				for _, dp := range data.DataPoints {
					if matchesAttributes(dp.Attributes, attrs, cfg.subset, encoder) {
						verifyValue(t, dp.Value, expectedValue, cfg.atLeast, metricName, attrs)
						return
					}
				}
			}
		}
	}

	require.True(t, foundMetric, "metric %s not found", metricName)
	require.Fail(t, "Data point for attributes %v not found in %s metric", attrs, metricName)
}

// VerifyHistogramMetric finds a histogram metric across all scopes and verifies its count.
// By default, it requires an exact attribute match and an exact count match.
// Use AtLeast() or Subset() options to relax these requirements.
func VerifyHistogramMetric(t *testing.T, ctx context.Context, reader *metric.ManualReader, metricName string, attrs attribute.Set, expectedCount uint64, options ...VerifyOption) {
	t.Helper()
	cfg := &verifyConfig{}
	for _, opt := range options {
		opt(cfg)
	}

	var rm metricdata.ResourceMetrics
	err := reader.Collect(ctx, &rm)
	require.NoError(t, err, "reader.Collect")
	encoder := attribute.DefaultEncoder()

	foundMetric := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricName {
				foundMetric = true
				switch data := m.Data.(type) {
				case metricdata.Histogram[int64]:
					for _, dp := range data.DataPoints {
						if matchesAttributes(dp.Attributes, attrs, cfg.subset, encoder) {
							verifyValue(t, dp.Count, expectedCount, cfg.atLeast, metricName, attrs)
							return
						}
					}
				case metricdata.Histogram[float64]:
					for _, dp := range data.DataPoints {
						if matchesAttributes(dp.Attributes, attrs, cfg.subset, encoder) {
							verifyValue(t, dp.Count, expectedCount, cfg.atLeast, metricName, attrs)
							return
						}
					}
				default:
					require.Fail(t, "metric %s is not an expected histogram type, but %T", metricName, m.Data)
				}
			}
		}
	}
	require.True(t, foundMetric, "metric %s not found", metricName)
	require.Fail(t, "Data point for attributes %v not found in %s metric", attrs, metricName)
}

// VerifyHistogramFull finds a histogram metric and fully verifies its state including total count, sum, and bucket distribution.
// expectedBuckets is a map of bucket indices to their expected counts.
func VerifyHistogramFull[T int64 | float64](t *testing.T, ctx context.Context, reader *metric.ManualReader, metricName string, attrs attribute.Set, expectedCount uint64, expectedSum T, expectedBuckets map[int]uint64, options ...VerifyOption) {
	t.Helper()
	cfg := &verifyConfig{}
	for _, opt := range options {
		opt(cfg)
	}

	var rm metricdata.ResourceMetrics
	err := reader.Collect(ctx, &rm)
	require.NoError(t, err, "reader.Collect")
	encoder := attribute.DefaultEncoder()

	foundMetric := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricName {
				foundMetric = true
				data, ok := m.Data.(metricdata.Histogram[T])
				require.True(t, ok, "metric %s is not of expected histogram type %T, but %T", metricName, data, m.Data)

				for _, dp := range data.DataPoints {
					if matchesAttributes(dp.Attributes, attrs, cfg.subset, encoder) {
						// Assert total count
						require.Equal(t, expectedCount, dp.Count, "Total count mismatch for %s", metricName)
						
						// Assert total sum
						if !cfg.atLeast {
							require.Equal(t, expectedSum, dp.Sum, "Total sum mismatch for %s", metricName)
						}

						// Assert individual bucket counts
						for bucketIdx, expBucketCount := range expectedBuckets {
							require.GreaterOrEqual(t, len(dp.BucketCounts), bucketIdx+1, "Bucket index %d out of range for %s", bucketIdx, metricName)
							require.Equal(t, expBucketCount, dp.BucketCounts[bucketIdx], "Bucket %d count mismatch for %s", bucketIdx, metricName)
						}

						// Verify that sum of all bucket counts matches total count
						var totalBucketCount uint64
						for _, count := range dp.BucketCounts {
							totalBucketCount += count
						}
						require.Equal(t, expectedCount, totalBucketCount, "Sum of bucket counts must equal total count for %s", metricName)
						return
					}
				}
			}
		}
	}
	require.True(t, foundMetric, "metric %s not found", metricName)
	require.Fail(t, "Data point for attributes %v not found in %s metric", attrs, metricName)
}
