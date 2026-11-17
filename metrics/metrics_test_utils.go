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
	"fmt"
	"reflect"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// TestingT is the subset of testing.TB required by metric verification helpers
// (satisfied by *testing.T, testing.TB, and *assert.CollectT in require.EventuallyWithT).
type TestingT interface {
	Errorf(format string, args ...any)
	FailNow()
	Helper()
}

// VerifyOption configures metric lookup and assertion behavior.
type VerifyOption func(*verifyConfig)

type verifyConfig struct {
	atLeast bool
	subset  bool
	rm      *metricdata.ResourceMetrics
}

// AtLeast checks that the metric value or count is greater than or equal to the
// expected value rather than requiring an exact match.
func AtLeast() VerifyOption {
	return func(c *verifyConfig) { c.atLeast = true }
}

// Subset checks that the provided attributes are a typed subset of the recorded
// data point attributes rather than requiring an exact set match.
func Subset() VerifyOption {
	return func(c *verifyConfig) { c.subset = true }
}

// CollectResourceMetrics collects a point-in-time ResourceMetrics snapshot from
// reader so multi-assertion tests can reuse one collection via WithResourceMetrics.
func CollectResourceMetrics(t TestingT, ctx context.Context, reader *metric.ManualReader) *metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm), "reader.Collect")
	return &rm
}

// WithResourceMetrics reuses a pre-collected ResourceMetrics snapshot instead
// of calling reader.Collect again on each assertion.
func WithResourceMetrics(rm *metricdata.ResourceMetrics) VerifyOption {
	return func(c *verifyConfig) { c.rm = rm }
}

// VerifyCounterMetric finds an int64 sum metric across scopes and verifies its
// value. By default it requires an exact attribute match and exact value match.
func VerifyCounterMetric(t TestingT, ctx context.Context, reader *metric.ManualReader, metricName string, attrs attribute.Set, expectedValue int64, options ...VerifyOption) {
	t.Helper()
	cfg := parseVerifyConfig(options...)
	matches := collectMatchingMetrics(t, ctx, reader, metricName, cfg)
	require.NotEmpty(t, matches, "metric %s not found", metricName)

	var seenAttrs []attribute.Set
	for _, m := range matches {
		data, ok := m.Data.(metricdata.Sum[int64])
		require.True(t, ok, "metric %s is not a Sum[int64], but %T", metricName, m.Data)

		for _, dp := range data.DataPoints {
			seenAttrs = append(seenAttrs, dp.Attributes)
			if matchesAttributes(dp.Attributes, attrs, cfg.subset) {
				verifyValue(t, dp.Value, expectedValue, cfg.atLeast, "value", metricName, attrs)
				return
			}
		}
	}

	require.Failf(t, "data point not found", "no data point matching attributes %v in metric %s (seen: %v)", attrs, metricName, seenAttrs)
}

// VerifyHistogramMetric finds an int64 or float64 histogram metric across
// scopes and verifies its sample count.
func VerifyHistogramMetric(t TestingT, ctx context.Context, reader *metric.ManualReader, metricName string, attrs attribute.Set, expectedCount uint64, options ...VerifyOption) {
	t.Helper()
	cfg := parseVerifyConfig(options...)
	matches := collectMatchingMetrics(t, ctx, reader, metricName, cfg)
	require.NotEmpty(t, matches, "metric %s not found", metricName)

	var seenAttrs []attribute.Set
	for _, m := range matches {
		switch data := m.Data.(type) {
		case metricdata.Histogram[int64]:
			if count, ok := findHistogramCount(data.DataPoints, attrs, cfg.subset, &seenAttrs); ok {
				verifyValue(t, count, expectedCount, cfg.atLeast, "count", metricName, attrs)
				return
			}
		case metricdata.Histogram[float64]:
			if count, ok := findHistogramCount(data.DataPoints, attrs, cfg.subset, &seenAttrs); ok {
				verifyValue(t, count, expectedCount, cfg.atLeast, "count", metricName, attrs)
				return
			}
		default:
			require.Failf(t, "unexpected histogram type", "metric %s is not an expected histogram type, but %T", metricName, m.Data)
		}
	}

	require.Failf(t, "data point not found", "no data point matching attributes %v in metric %s (seen: %v)", attrs, metricName, seenAttrs)
}

// VerifyHistogramFull finds a typed histogram metric (int64 or float64) across
// scopes and verifies its count, sum, bucket total invariant, and any bucket
// counts specified in expectedBuckets (pass nil to check count and sum only).
func VerifyHistogramFull[T int64 | float64](t TestingT, ctx context.Context, reader *metric.ManualReader, metricName string, attrs attribute.Set, expectedCount uint64, expectedSum T, expectedBuckets map[int]uint64, options ...VerifyOption) {
	t.Helper()
	cfg := parseVerifyConfig(options...)
	matches := collectMatchingMetrics(t, ctx, reader, metricName, cfg)
	require.NotEmpty(t, matches, "metric %s not found", metricName)

	var (
		dp        metricdata.HistogramDataPoint[T]
		foundDP   bool
		seenAttrs []attribute.Set
	)
	for _, m := range matches {
		data, ok := m.Data.(metricdata.Histogram[T])
		require.True(t, ok, "metric %s is not of expected histogram type %T, but %T", metricName, data, m.Data)
		for _, cand := range data.DataPoints {
			seenAttrs = append(seenAttrs, cand.Attributes)
			if matchesAttributes(cand.Attributes, attrs, cfg.subset) {
				dp = cand
				foundDP = true
				break
			}
		}
		if foundDP {
			break
		}
	}
	if !foundDP {
		require.Failf(t, "data point not found", "no data point matching attributes %v in metric %s (seen: %v)", attrs, metricName, seenAttrs)
	}

	verifyValue(t, dp.Count, expectedCount, cfg.atLeast, "count", metricName, attrs)
	verifyValue(t, dp.Sum, expectedSum, cfg.atLeast, "sum", metricName, attrs)

	for bucketIdx, expBucketCount := range expectedBuckets {
		require.True(t, bucketIdx >= 0 && bucketIdx < len(dp.BucketCounts), "Bucket index %d out of range [0, %d) for %s", bucketIdx, len(dp.BucketCounts), metricName)
		verifyValue(t, dp.BucketCounts[bucketIdx], expBucketCount, cfg.atLeast, fmt.Sprintf("bucket[%d]", bucketIdx), metricName, attrs)
	}

	var totalBucketCount uint64
	for _, count := range dp.BucketCounts {
		totalBucketCount += count
	}
	require.Equal(t, dp.Count, totalBucketCount, "Sum of bucket counts must equal total count for %s", metricName)
}

func parseVerifyConfig(options ...VerifyOption) *verifyConfig {
	cfg := &verifyConfig{}
	for _, opt := range options {
		opt(cfg)
	}
	return cfg
}

func collectMatchingMetrics(t TestingT, ctx context.Context, reader *metric.ManualReader, metricName string, cfg *verifyConfig) []metricdata.Metrics {
	t.Helper()
	rm := cfg.rm
	if rm == nil {
		rm = CollectResourceMetrics(t, ctx, reader)
	}

	var matches []metricdata.Metrics
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricName {
				matches = append(matches, m)
			}
		}
	}
	return matches
}

func findHistogramCount[T int64 | float64](dps []metricdata.HistogramDataPoint[T], attrs attribute.Set, subset bool, seenAttrs *[]attribute.Set) (uint64, bool) {
	for _, dp := range dps {
		*seenAttrs = append(*seenAttrs, dp.Attributes)
		if matchesAttributes(dp.Attributes, attrs, subset) {
			return dp.Count, true
		}
	}
	return 0, false
}

func matchesAttributes(dpAttrs, targetAttrs attribute.Set, subset bool) bool {
	if !subset {
		return dpAttrs.Equals(&targetAttrs)
	}
	for iter := targetAttrs.Iter(); iter.Next(); {
		targetKV := iter.Attribute()
		val, ok := dpAttrs.Value(targetKV.Key)
		if !ok || val.Type() != targetKV.Value.Type() {
			return false
		}
		switch val.Type() {
		case attribute.BOOL, attribute.INT64, attribute.FLOAT64, attribute.STRING:
			if val != targetKV.Value {
				return false
			}
		default:
			if !reflect.DeepEqual(val.AsInterface(), targetKV.Value.AsInterface()) {
				return false
			}
		}
	}
	return true
}

func verifyValue[T int64 | uint64 | float64](t TestingT, actual, expected T, atLeast bool, field, metricName string, attrs attribute.Set) {
	t.Helper()
	if atLeast {
		assert.GreaterOrEqual(t, actual, expected, "metric %s too low for %s with attributes %v", field, metricName, attrs)
	} else {
		assert.Equal(t, expected, actual, "metric %s mismatch for %s with attributes %v", field, metricName, attrs)
	}
}
