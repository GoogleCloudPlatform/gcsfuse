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
	"time"

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

	encoder := attribute.DefaultEncoder()

	require.Eventually(t, func() bool {
		var rm metricdata.ResourceMetrics
		if err := reader.Collect(ctx, &rm); err != nil {
			return false
		}
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name == metricName {
					if data, ok := m.Data.(metricdata.Sum[int64]); ok {
						for _, dp := range data.DataPoints {
							if matchesAttributes(dp.Attributes, attrs, cfg.subset, encoder) {
								if cfg.atLeast {
									return dp.Value >= expectedValue
								}
								return dp.Value == expectedValue
							}
						}
					}
				}
			}
		}
		return false
	}, 2*time.Second, 10*time.Millisecond, "counter metric %s with attrs %v and expected value %d not found or value mismatch", metricName, attrs, expectedValue)
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

	encoder := attribute.DefaultEncoder()

	require.Eventually(t, func() bool {
		var rm metricdata.ResourceMetrics
		if err := reader.Collect(ctx, &rm); err != nil {
			return false
		}
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name == metricName {
					switch data := m.Data.(type) {
					case metricdata.Histogram[int64]:
						for _, dp := range data.DataPoints {
							if matchesAttributes(dp.Attributes, attrs, cfg.subset, encoder) {
								if cfg.atLeast {
									return dp.Count >= expectedCount
								}
								return dp.Count == expectedCount
							}
						}
					case metricdata.Histogram[float64]:
						for _, dp := range data.DataPoints {
							if matchesAttributes(dp.Attributes, attrs, cfg.subset, encoder) {
								if cfg.atLeast {
									return dp.Count >= expectedCount
								}
								return dp.Count == expectedCount
							}
						}
					}
				}
			}
		}
		return false
	}, 2*time.Second, 10*time.Millisecond, "histogram metric %s with attrs %v and expected count %d not found or count mismatch", metricName, attrs, expectedCount)
}

// VerifyHistogramFull finds a histogram metric and fully verifies its state including total count, sum, and bucket distribution.
// expectedBuckets is a map of bucket indices to their expected counts.
func VerifyHistogramFull[T int64 | float64](t *testing.T, ctx context.Context, reader *metric.ManualReader, metricName string, attrs attribute.Set, expectedCount uint64, expectedSum T, expectedBuckets map[int]uint64, options ...VerifyOption) {
	t.Helper()
	cfg := &verifyConfig{}
	for _, opt := range options {
		opt(cfg)
	}

	encoder := attribute.DefaultEncoder()

	require.Eventually(t, func() bool {
		var rm metricdata.ResourceMetrics
		if err := reader.Collect(ctx, &rm); err != nil {
			return false
		}
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name == metricName {
					if data, ok := m.Data.(metricdata.Histogram[T]); ok {
						for _, dp := range data.DataPoints {
							if matchesAttributes(dp.Attributes, attrs, cfg.subset, encoder) {
								if dp.Count != expectedCount {
									return false
								}
								if !cfg.atLeast && dp.Sum != expectedSum {
									return false
								}
								for bucketIdx, expBucketCount := range expectedBuckets {
									if len(dp.BucketCounts) < bucketIdx+1 || dp.BucketCounts[bucketIdx] != expBucketCount {
										return false
									}
								}
								var totalBucketCount uint64
								for _, count := range dp.BucketCounts {
									totalBucketCount += count
								}
								return totalBucketCount == expectedCount
							}
						}
					}
				}
			}
		}
		return false
	}, 2*time.Second, 10*time.Millisecond, "histogram metric %s with attrs %v not found or full verification failed", metricName, attrs)
}
