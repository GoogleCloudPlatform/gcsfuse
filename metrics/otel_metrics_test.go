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

package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// metricValueMap maps attribute sets to metric values.
type metricValueMap map[string]int64

// metricHistogramMap maps attribute sets to histogram data points.
type metricHistogramMap map[string]metricdata.HistogramDataPoint[int64]

func waitForMetricsProcessing() {
	time.Sleep(time.Millisecond)
}

func setupOTel(ctx context.Context, t *testing.T) (*otelMetrics, *metric.ManualReader) {
	t.Helper()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)

	m, err := NewOTelMetrics(ctx, 10, 100)
	require.NoError(t, err)
	return m, reader
}

// gatherHistogramMetrics collects all histogram metrics from the reader.
// It returns a map where the key is the metric name, and the value is another map.
// The inner map's key is a string representation of the attributes,
// and the value is the metricdata.HistogramDataPoint.
func gatherHistogramMetrics(ctx context.Context, t *testing.T, rd *metric.ManualReader) map[string]map[string]metricdata.HistogramDataPoint[int64] {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := rd.Collect(ctx, &rm)
	require.NoError(t, err)

	results := make(map[string]map[string]metricdata.HistogramDataPoint[int64])
	encoder := attribute.DefaultEncoder() // Using default encoder

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			// We are interested in Histogram[int64].
			hist, ok := m.Data.(metricdata.Histogram[int64])
			if !ok {
				continue
			}

			metricMap := make(metricHistogramMap)
			for _, dp := range hist.DataPoints {
				if dp.Count == 0 {
					continue
				}

				metricMap[dp.Attributes.Encoded(encoder)] = dp
			}

			if len(metricMap) > 0 {
				results[m.Name] = metricMap
			}
		}
	}

	return results
}

// gatherNonZeroCounterMetrics collects all non-zero counter metrics from the reader.
// It returns a map where the key is the metric name, and the value is another map.
// The inner map's key is a string representation of the attributes,
// and the value is the metric's value.
func gatherNonZeroCounterMetrics(ctx context.Context, t *testing.T, rd *metric.ManualReader) map[string]map[string]int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := rd.Collect(ctx, &rm)
	require.NoError(t, err)

	results := make(map[string]map[string]int64)
	encoder := attribute.DefaultEncoder()

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			// We are interested in Sum[int64] which corresponds to int_counter.
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}

			metricMap := make(metricValueMap)
			for _, dp := range sum.DataPoints {
				if dp.Value == 0 {
					continue
				}

				metricMap[dp.Attributes.Encoded(encoder)] = dp.Value
			}

			if len(metricMap) > 0 {
				results[m.Name] = metricMap
			}
		}
	}

	return results
}

func TestBufferedReadFallbackTriggerCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "reason_insufficient_memory",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(5, "insufficient_memory")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("reason", "insufficient_memory")): 5,
			},
		},
		{
			name: "reason_random_read_detected",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(5, "random_read_detected")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("reason", "random_read_detected")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(5, "insufficient_memory")
				m.BufferedReadFallbackTriggerCount(2, "random_read_detected")
				m.BufferedReadFallbackTriggerCount(3, "insufficient_memory")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("reason", "insufficient_memory")): 8,
				attribute.NewSet(attribute.String("reason", "random_read_detected")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(-5, "insufficient_memory")
				m.BufferedReadFallbackTriggerCount(2, "insufficient_memory")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("reason", "insufficient_memory")): 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			metric, ok := metrics["buffered_read/fallback_trigger_count"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "buffered_read/fallback_trigger_count metric should not be found")
				return
			}
			require.True(t, ok, "buffered_read/fallback_trigger_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, metric)
		})
	}
}

func TestBufferedReadReadLatency(t *testing.T) {
	ctx := context.Background()
	encoder := attribute.DefaultEncoder()
	m, rd := setupOTel(ctx, t)
	var totalLatency time.Duration
	latencies := []time.Duration{100 * time.Microsecond, 200 * time.Microsecond}

	for _, latency := range latencies {
		m.BufferedReadReadLatency(ctx, latency)
		totalLatency += latency
	}
	waitForMetricsProcessing()

	metrics := gatherHistogramMetrics(ctx, t, rd)
	metric, ok := metrics["buffered_read/read_latency"]
	require.True(t, ok, "buffered_read/read_latency metric not found")

	s := attribute.NewSet()
	expectedKey := s.Encoded(encoder)
	dp, ok := metric[expectedKey]
	require.True(t, ok, "DataPoint not found for key: %s", expectedKey)
	assert.Equal(t, uint64(len(latencies)), dp.Count)
	assert.Equal(t, totalLatency.Microseconds(), dp.Sum)
}

func TestFileCacheReadBytesCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "read_type_Parallel",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, "Parallel")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, "Random")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "read_type_Unknown",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, "Unknown")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Unknown")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, "Parallel")
				m.FileCacheReadBytesCount(2, "Random")
				m.FileCacheReadBytesCount(3, "Parallel")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(-5, "Parallel")
				m.FileCacheReadBytesCount(2, "Parallel")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Parallel")): 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			metric, ok := metrics["file_cache/read_bytes_count"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "file_cache/read_bytes_count metric should not be found")
				return
			}
			require.True(t, ok, "file_cache/read_bytes_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, metric)
		})
	}
}

func TestFileCacheReadCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "cache_hit_true_read_type_Parallel",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, "Parallel")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "cache_hit_true_read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, "Random")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "cache_hit_true_read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "cache_hit_true_read_type_Unknown",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, "Unknown")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Unknown")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Parallel",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, "Parallel")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, "Random")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Unknown",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, "Unknown")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Unknown")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, "Parallel")
				m.FileCacheReadCount(2, true, "Random")
				m.FileCacheReadCount(3, true, "Parallel")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(-5, true, "Parallel")
				m.FileCacheReadCount(2, true, "Parallel")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Parallel")): 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			metric, ok := metrics["file_cache/read_count"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "file_cache/read_count metric should not be found")
				return
			}
			require.True(t, ok, "file_cache/read_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, metric)
		})
	}
}

func TestFileCacheReadLatencies(t *testing.T) {
	tests := []struct {
		name      string
		latencies []time.Duration
		cacheHit  bool
	}{
		{
			name:      "cache_hit_true",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			cacheHit:  true,
		},
		{
			name:      "cache_hit_false",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			cacheHit:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)
			var totalLatency time.Duration

			for _, latency := range tc.latencies {
				m.FileCacheReadLatencies(ctx, latency, tc.cacheHit)
				totalLatency += latency
			}
			waitForMetricsProcessing()

			metrics := gatherHistogramMetrics(ctx, t, rd)
			metric, ok := metrics["file_cache/read_latencies"]
			require.True(t, ok, "file_cache/read_latencies metric not found")

			attrs := []attribute.KeyValue{
				attribute.Bool("cache_hit", tc.cacheHit),
			}
			s := attribute.NewSet(attrs...)
			expectedKey := s.Encoded(encoder)
			dp, ok := metric[expectedKey]
			require.True(t, ok, "DataPoint not found for key: %s", expectedKey)
			assert.Equal(t, uint64(len(tc.latencies)), dp.Count)
			assert.Equal(t, totalLatency.Microseconds(), dp.Sum)
		})
	}
}

func TestFsOpsCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "WriteFile")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "BatchForget")
				m.FsOpsCount(2, "CreateFile")
				m.FsOpsCount(3, "BatchForget")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_op", "BatchForget")): 8,
				attribute.NewSet(attribute.String("fs_op", "CreateFile")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FsOpsCount(-5, "BatchForget")
				m.FsOpsCount(2, "BatchForget")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_op", "BatchForget")): 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			metric, ok := metrics["fs/ops_count"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "fs/ops_count metric should not be found")
				return
			}
			require.True(t, ok, "fs/ops_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, metric)
		})
	}
}

func TestFsOpsErrorCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DIR_NOT_EMPTY", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_DIR_ERROR", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "FILE_EXISTS", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INTERRUPT_ERROR", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_ARGUMENT", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "INVALID_OPERATION", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "IO_ERROR", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "MISC_ERROR", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NETWORK_ERROR", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_A_DIR", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NOT_IMPLEMENTED", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "NO_FILE_OR_DIR", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PERM_ERROR", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "PROCESS_RESOURCE_MGMT_ERROR", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Others",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "Others")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Others")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "ReadDirPlus")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "TOO_MANY_OPEN_FILES", "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "WriteFile")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, "DEVICE_ERROR", "BatchForget")
				m.FsOpsErrorCount(2, "DEVICE_ERROR", "CreateFile")
				m.FsOpsErrorCount(3, "DEVICE_ERROR", "BatchForget")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "BatchForget")): 8,
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateFile")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(-5, "DEVICE_ERROR", "BatchForget")
				m.FsOpsErrorCount(2, "DEVICE_ERROR", "BatchForget")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "BatchForget")): 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			metric, ok := metrics["fs/ops_error_count"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "fs/ops_error_count metric should not be found")
				return
			}
			require.True(t, ok, "fs/ops_error_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, metric)
		})
	}
}

func TestFsOpsLatency(t *testing.T) {
	tests := []struct {
		name      string
		latencies []time.Duration
		fsOp      FsOp
	}{
		{
			name:      "fs_op_BatchForget",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "BatchForget",
		},
		{
			name:      "fs_op_CreateFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "CreateFile",
		},
		{
			name:      "fs_op_CreateLink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "CreateLink",
		},
		{
			name:      "fs_op_CreateSymlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "CreateSymlink",
		},
		{
			name:      "fs_op_FlushFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "FlushFile",
		},
		{
			name:      "fs_op_ForgetInode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "ForgetInode",
		},
		{
			name:      "fs_op_GetInodeAttributes",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "GetInodeAttributes",
		},
		{
			name:      "fs_op_LookUpInode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "LookUpInode",
		},
		{
			name:      "fs_op_MkDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "MkDir",
		},
		{
			name:      "fs_op_MkNode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "MkNode",
		},
		{
			name:      "fs_op_OpenDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "OpenDir",
		},
		{
			name:      "fs_op_OpenFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "OpenFile",
		},
		{
			name:      "fs_op_Others",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "Others",
		},
		{
			name:      "fs_op_ReadDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "ReadDir",
		},
		{
			name:      "fs_op_ReadDirPlus",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "ReadDirPlus",
		},
		{
			name:      "fs_op_ReadFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "ReadFile",
		},
		{
			name:      "fs_op_ReadSymlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "ReadSymlink",
		},
		{
			name:      "fs_op_ReleaseDirHandle",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "ReleaseDirHandle",
		},
		{
			name:      "fs_op_ReleaseFileHandle",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "ReleaseFileHandle",
		},
		{
			name:      "fs_op_Rename",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "Rename",
		},
		{
			name:      "fs_op_RmDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "RmDir",
		},
		{
			name:      "fs_op_SetInodeAttributes",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "SetInodeAttributes",
		},
		{
			name:      "fs_op_SyncFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "SyncFile",
		},
		{
			name:      "fs_op_Unlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "Unlink",
		},
		{
			name:      "fs_op_WriteFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      "WriteFile",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)
			var totalLatency time.Duration

			for _, latency := range tc.latencies {
				m.FsOpsLatency(ctx, latency, tc.fsOp)
				totalLatency += latency
			}
			waitForMetricsProcessing()

			metrics := gatherHistogramMetrics(ctx, t, rd)
			metric, ok := metrics["fs/ops_latency"]
			require.True(t, ok, "fs/ops_latency metric not found")

			attrs := []attribute.KeyValue{
				attribute.String("fs_op", string(tc.fsOp)),
			}
			s := attribute.NewSet(attrs...)
			expectedKey := s.Encoded(encoder)
			dp, ok := metric[expectedKey]
			require.True(t, ok, "DataPoint not found for key: %s", expectedKey)
			assert.Equal(t, uint64(len(tc.latencies)), dp.Count)
			assert.Equal(t, totalLatency.Microseconds(), dp.Sum)
		})
	}
}

func TestGcsDownloadBytesCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "read_type_Buffered",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, "Buffered")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Buffered")): 5,
			},
		},
		{
			name: "read_type_Parallel",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, "Parallel")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, "Random")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, "Buffered")
				m.GcsDownloadBytesCount(2, "Parallel")
				m.GcsDownloadBytesCount(3, "Buffered")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Buffered")): 8,
				attribute.NewSet(attribute.String("read_type", "Parallel")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(-5, "Buffered")
				m.GcsDownloadBytesCount(2, "Buffered")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Buffered")): 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			metric, ok := metrics["gcs/download_bytes_count"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "gcs/download_bytes_count metric should not be found")
				return
			}
			require.True(t, ok, "gcs/download_bytes_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, metric)
		})
	}
}

func TestGcsReadBytesCount(t *testing.T) {
	ctx := context.Background()
	encoder := attribute.DefaultEncoder()
	m, rd := setupOTel(ctx, t)

	m.GcsReadBytesCount(1024)
	m.GcsReadBytesCount(2048)
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	metric, ok := metrics["gcs/read_bytes_count"]
	require.True(t, ok, "gcs/read_bytes_count metric not found")
	s := attribute.NewSet()
	assert.Equal(t, map[string]int64{s.Encoded(encoder): 3072}, metric, "Positive increments should be summed.")

	// Test negative increment
	m.GcsReadBytesCount(-100)
	waitForMetricsProcessing()

	metrics = gatherNonZeroCounterMetrics(ctx, t, rd)
	metric, ok = metrics["gcs/read_bytes_count"]
	require.True(t, ok, "gcs/read_bytes_count metric not found after negative increment")
	assert.Equal(t, map[string]int64{s.Encoded(encoder): 3072}, metric, "Negative increment should not change the metric value.")
}

func TestGcsReadCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "read_type_Parallel",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, "Parallel")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, "Random")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "read_type_Unknown",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, "Unknown")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Unknown")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, "Parallel")
				m.GcsReadCount(2, "Random")
				m.GcsReadCount(3, "Parallel")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsReadCount(-5, "Parallel")
				m.GcsReadCount(2, "Parallel")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Parallel")): 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			metric, ok := metrics["gcs/read_count"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "gcs/read_count metric should not be found")
				return
			}
			require.True(t, ok, "gcs/read_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, metric)
		})
	}
}

func TestGcsReaderCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "io_method_ReadHandle",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, "ReadHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "ReadHandle")): 5,
			},
		},
		{
			name: "io_method_closed",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, "closed")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "closed")): 5,
			},
		},
		{
			name: "io_method_opened",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, "opened")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "opened")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, "ReadHandle")
				m.GcsReaderCount(2, "closed")
				m.GcsReaderCount(3, "ReadHandle")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("io_method", "ReadHandle")): 8,
				attribute.NewSet(attribute.String("io_method", "closed")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(-5, "ReadHandle")
				m.GcsReaderCount(2, "ReadHandle")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("io_method", "ReadHandle")): 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			metric, ok := metrics["gcs/reader_count"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "gcs/reader_count metric should not be found")
				return
			}
			require.True(t, ok, "gcs/reader_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, metric)
		})
	}
}

func TestGcsRequestCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "gcs_method_ComposeObjects",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "ComposeObjects")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "ComposeObjects")): 5,
			},
		},
		{
			name: "gcs_method_CopyObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "CopyObject")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CopyObject")): 5,
			},
		},
		{
			name: "gcs_method_CreateAppendableObjectWriter",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "CreateAppendableObjectWriter")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateAppendableObjectWriter")): 5,
			},
		},
		{
			name: "gcs_method_CreateFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "CreateFolder")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateFolder")): 5,
			},
		},
		{
			name: "gcs_method_CreateObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "CreateObject")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateObject")): 5,
			},
		},
		{
			name: "gcs_method_CreateObjectChunkWriter",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "CreateObjectChunkWriter")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateObjectChunkWriter")): 5,
			},
		},
		{
			name: "gcs_method_DeleteFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "DeleteFolder")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "DeleteFolder")): 5,
			},
		},
		{
			name: "gcs_method_DeleteObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "DeleteObject")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "DeleteObject")): 5,
			},
		},
		{
			name: "gcs_method_FinalizeUpload",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "FinalizeUpload")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "FinalizeUpload")): 5,
			},
		},
		{
			name: "gcs_method_FlushPendingWrites",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "FlushPendingWrites")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "FlushPendingWrites")): 5,
			},
		},
		{
			name: "gcs_method_GetFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "GetFolder")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "GetFolder")): 5,
			},
		},
		{
			name: "gcs_method_ListObjects",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "ListObjects")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "ListObjects")): 5,
			},
		},
		{
			name: "gcs_method_MoveObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "MoveObject")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "MoveObject")): 5,
			},
		},
		{
			name: "gcs_method_MultiRangeDownloader::Add",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "MultiRangeDownloader::Add")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "MultiRangeDownloader::Add")): 5,
			},
		},
		{
			name: "gcs_method_NewMultiRangeDownloader",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "NewMultiRangeDownloader")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "NewMultiRangeDownloader")): 5,
			},
		},
		{
			name: "gcs_method_NewReader",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "NewReader")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "NewReader")): 5,
			},
		},
		{
			name: "gcs_method_RenameFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "RenameFolder")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "RenameFolder")): 5,
			},
		},
		{
			name: "gcs_method_StatObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "StatObject")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "StatObject")): 5,
			},
		},
		{
			name: "gcs_method_UpdateObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "UpdateObject")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "UpdateObject")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, "ComposeObjects")
				m.GcsRequestCount(2, "CopyObject")
				m.GcsRequestCount(3, "ComposeObjects")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("gcs_method", "ComposeObjects")): 8,
				attribute.NewSet(attribute.String("gcs_method", "CopyObject")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(-5, "ComposeObjects")
				m.GcsRequestCount(2, "ComposeObjects")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("gcs_method", "ComposeObjects")): 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			metric, ok := metrics["gcs/request_count"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "gcs/request_count metric should not be found")
				return
			}
			require.True(t, ok, "gcs/request_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, metric)
		})
	}
}

func TestGcsRequestLatencies(t *testing.T) {
	tests := []struct {
		name      string
		latencies []time.Duration
		gcsMethod GcsMethod
	}{
		{
			name:      "gcs_method_ComposeObjects",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "ComposeObjects",
		},
		{
			name:      "gcs_method_CopyObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "CopyObject",
		},
		{
			name:      "gcs_method_CreateAppendableObjectWriter",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "CreateAppendableObjectWriter",
		},
		{
			name:      "gcs_method_CreateFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "CreateFolder",
		},
		{
			name:      "gcs_method_CreateObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "CreateObject",
		},
		{
			name:      "gcs_method_CreateObjectChunkWriter",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "CreateObjectChunkWriter",
		},
		{
			name:      "gcs_method_DeleteFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "DeleteFolder",
		},
		{
			name:      "gcs_method_DeleteObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "DeleteObject",
		},
		{
			name:      "gcs_method_FinalizeUpload",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "FinalizeUpload",
		},
		{
			name:      "gcs_method_FlushPendingWrites",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "FlushPendingWrites",
		},
		{
			name:      "gcs_method_GetFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "GetFolder",
		},
		{
			name:      "gcs_method_ListObjects",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "ListObjects",
		},
		{
			name:      "gcs_method_MoveObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "MoveObject",
		},
		{
			name:      "gcs_method_MultiRangeDownloader::Add",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "MultiRangeDownloader::Add",
		},
		{
			name:      "gcs_method_NewMultiRangeDownloader",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "NewMultiRangeDownloader",
		},
		{
			name:      "gcs_method_NewReader",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "NewReader",
		},
		{
			name:      "gcs_method_RenameFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "RenameFolder",
		},
		{
			name:      "gcs_method_StatObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "StatObject",
		},
		{
			name:      "gcs_method_UpdateObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: "UpdateObject",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)
			var totalLatency time.Duration

			for _, latency := range tc.latencies {
				m.GcsRequestLatencies(ctx, latency, tc.gcsMethod)
				totalLatency += latency
			}
			waitForMetricsProcessing()

			metrics := gatherHistogramMetrics(ctx, t, rd)
			metric, ok := metrics["gcs/request_latencies"]
			require.True(t, ok, "gcs/request_latencies metric not found")

			attrs := []attribute.KeyValue{
				attribute.String("gcs_method", string(tc.gcsMethod)),
			}
			s := attribute.NewSet(attrs...)
			expectedKey := s.Encoded(encoder)
			dp, ok := metric[expectedKey]
			require.True(t, ok, "DataPoint not found for key: %s", expectedKey)
			assert.Equal(t, uint64(len(tc.latencies)), dp.Count)
			assert.Equal(t, totalLatency.Milliseconds(), dp.Sum)
		})
	}
}

func TestGcsRetryCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "retry_error_category_OTHER_ERRORS",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(5, "OTHER_ERRORS")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", "OTHER_ERRORS")): 5,
			},
		},
		{
			name: "retry_error_category_STALLED_READ_REQUEST",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(5, "STALLED_READ_REQUEST")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", "STALLED_READ_REQUEST")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(5, "OTHER_ERRORS")
				m.GcsRetryCount(2, "STALLED_READ_REQUEST")
				m.GcsRetryCount(3, "OTHER_ERRORS")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("retry_error_category", "OTHER_ERRORS")): 8,
				attribute.NewSet(attribute.String("retry_error_category", "STALLED_READ_REQUEST")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(-5, "OTHER_ERRORS")
				m.GcsRetryCount(2, "OTHER_ERRORS")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("retry_error_category", "OTHER_ERRORS")): 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			metric, ok := metrics["gcs/retry_count"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "gcs/retry_count metric should not be found")
				return
			}
			require.True(t, ok, "gcs/retry_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, metric)
		})
	}
}

func TestTestUpdownCounter(t *testing.T) {
	ctx := context.Background()
	encoder := attribute.DefaultEncoder()
	m, rd := setupOTel(ctx, t)

	m.TestUpdownCounter(1024)
	m.TestUpdownCounter(2048)
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	metric, ok := metrics["test/updown_counter"]
	require.True(t, ok, "test/updown_counter metric not found")
	s := attribute.NewSet()
	assert.Equal(t, map[string]int64{s.Encoded(encoder): 3072}, metric, "Positive increments should be summed.")

	// Test negative increment
	m.TestUpdownCounter(-100)
	waitForMetricsProcessing()

	metrics = gatherNonZeroCounterMetrics(ctx, t, rd)
	metric, ok = metrics["test/updown_counter"]
	require.True(t, ok, "test/updown_counter metric not found after negative increment")
	assert.Equal(t, map[string]int64{s.Encoded(encoder): 2972}, metric, "Negative increment should change the metric value.")
}

func TestTestUpdownCounterWithAttrs(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "request_type_attr1",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(5, "attr1")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("request_type", "attr1")): 5,
			},
		},
		{
			name: "request_type_attr2",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(5, "attr2")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("request_type", "attr2")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(5, "attr1")
				m.TestUpdownCounterWithAttrs(2, "attr2")
				m.TestUpdownCounterWithAttrs(3, "attr1")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("request_type", "attr1")): 8,
				attribute.NewSet(attribute.String("request_type", "attr2")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(-5, "attr1")
				m.TestUpdownCounterWithAttrs(2, "attr1")
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("request_type", "attr1")): -3},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			metric, ok := metrics["test/updown_counter_with_attrs"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "test/updown_counter_with_attrs metric should not be found")
				return
			}
			require.True(t, ok, "test/updown_counter_with_attrs metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, metric)
		})
	}
}
