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

func TestBufferedReadDownloadBlockLatency(t *testing.T) {
	tests := []struct {
		name      string
		latencies []time.Duration
		status    Status
	}{
		{
			name:      "status_cancelled",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			status:    Status("cancelled"),
		},
		{
			name:      "status_failed",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			status:    Status("failed"),
		},
		{
			name:      "status_successful",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			status:    Status("successful"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)
			var totalLatency time.Duration

			for _, latency := range tc.latencies {
				m.BufferedReadDownloadBlockLatency(ctx, latency, tc.status)
				totalLatency += latency
			}
			waitForMetricsProcessing()

			metrics := gatherHistogramMetrics(ctx, t, rd)
			metric, ok := metrics["buffered_read/download_block_latency"]
			require.True(t, ok, "buffered_read/download_block_latency metric not found")

			attrs := []attribute.KeyValue{
				attribute.String("status", string(tc.status)),
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

func TestBufferedReadFallbackTriggerCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "reason_insufficient_memory",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(5, Reason("insufficient_memory"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("reason", "insufficient_memory")): 5,
			},
		},
		{
			name: "reason_random_read_detected",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(5, Reason("random_read_detected"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("reason", "random_read_detected")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(5, Reason("insufficient_memory"))
				m.BufferedReadFallbackTriggerCount(2, Reason("random_read_detected"))
				m.BufferedReadFallbackTriggerCount(3, Reason("insufficient_memory"))
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("reason", "insufficient_memory")): 8,
				attribute.NewSet(attribute.String("reason", "random_read_detected")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(-5, Reason("insufficient_memory"))
				m.BufferedReadFallbackTriggerCount(2, Reason("insufficient_memory"))
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

func TestBufferedReadScheduledBlockCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "status_cancelled",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(5, Status("cancelled"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("status", "cancelled")): 5,
			},
		},
		{
			name: "status_failed",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(5, Status("failed"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("status", "failed")): 5,
			},
		},
		{
			name: "status_successful",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(5, Status("successful"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("status", "successful")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(5, Status("cancelled"))
				m.BufferedReadScheduledBlockCount(2, Status("failed"))
				m.BufferedReadScheduledBlockCount(3, Status("cancelled"))
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("status", "cancelled")): 8,
				attribute.NewSet(attribute.String("status", "failed")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(-5, Status("cancelled"))
				m.BufferedReadScheduledBlockCount(2, Status("cancelled"))
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("status", "cancelled")): 2},
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
			metric, ok := metrics["buffered_read/scheduled_block_count"]
			if len(tc.expected) == 0 {
				assert.False(t, ok, "buffered_read/scheduled_block_count metric should not be found")
				return
			}
			require.True(t, ok, "buffered_read/scheduled_block_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, metric)
		})
	}
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
				m.FileCacheReadBytesCount(5, ReadType("Parallel"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, ReadType("Random"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, ReadType("Sequential"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "read_type_Unknown",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, ReadType("Unknown"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Unknown")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, ReadType("Parallel"))
				m.FileCacheReadBytesCount(2, ReadType("Random"))
				m.FileCacheReadBytesCount(3, ReadType("Parallel"))
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(-5, ReadType("Parallel"))
				m.FileCacheReadBytesCount(2, ReadType("Parallel"))
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
				m.FileCacheReadCount(5, true, ReadType("Parallel"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "cache_hit_true_read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, ReadType("Random"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "cache_hit_true_read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, ReadType("Sequential"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "cache_hit_true_read_type_Unknown",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, ReadType("Unknown"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Unknown")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Parallel",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, ReadType("Parallel"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, ReadType("Random"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, ReadType("Sequential"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Unknown",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, ReadType("Unknown"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Unknown")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, ReadType("Parallel"))
				m.FileCacheReadCount(2, true, ReadType("Random"))
				m.FileCacheReadCount(3, true, ReadType("Parallel"))
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(-5, true, ReadType("Parallel"))
				m.FileCacheReadCount(2, true, ReadType("Parallel"))
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
				m.FsOpsCount(5, FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "WriteFile")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOp("BatchForget"))
				m.FsOpsCount(2, FsOp("CreateFile"))
				m.FsOpsCount(3, FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_op", "BatchForget")): 8,
				attribute.NewSet(attribute.String("fs_op", "CreateFile")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FsOpsCount(-5, FsOp("BatchForget"))
				m.FsOpsCount(2, FsOp("BatchForget"))
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
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DIR_NOT_EMPTY"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_DIR_ERROR"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("FILE_EXISTS"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INTERRUPT_ERROR"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_ARGUMENT"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("INVALID_OPERATION"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("IO_ERROR"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("MISC_ERROR"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NETWORK_ERROR"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_A_DIR"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NOT_IMPLEMENTED"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("NO_FILE_OR_DIR"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PERM_ERROR"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("PROCESS_RESOURCE_MGMT_ERROR"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("CreateFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("CreateLink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("CreateSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("Fallocate"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("FlushFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("ForgetInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("GetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("GetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("ListXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("LookUpInode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("MkDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("MkNode"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("OpenDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("OpenFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("ReadDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("ReadDirPlus"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("ReadFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("ReadSymlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("ReleaseDirHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("ReleaseFileHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("RemoveXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("Rename"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("RmDir"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("SetInodeAttributes"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("SetXattr"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("StatFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("SyncFS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("SyncFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("Unlink"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("TOO_MANY_OPEN_FILES"), FsOp("WriteFile"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "WriteFile")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategory("DEVICE_ERROR"), FsOp("BatchForget"))
				m.FsOpsErrorCount(2, FsErrorCategory("DEVICE_ERROR"), FsOp("CreateFile"))
				m.FsOpsErrorCount(3, FsErrorCategory("DEVICE_ERROR"), FsOp("BatchForget"))
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "BatchForget")): 8,
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateFile")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(-5, FsErrorCategory("DEVICE_ERROR"), FsOp("BatchForget"))
				m.FsOpsErrorCount(2, FsErrorCategory("DEVICE_ERROR"), FsOp("BatchForget"))
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
			fsOp:      FsOp("BatchForget"),
		},
		{
			name:      "fs_op_CreateFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("CreateFile"),
		},
		{
			name:      "fs_op_CreateLink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("CreateLink"),
		},
		{
			name:      "fs_op_CreateSymlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("CreateSymlink"),
		},
		{
			name:      "fs_op_Fallocate",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("Fallocate"),
		},
		{
			name:      "fs_op_FlushFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("FlushFile"),
		},
		{
			name:      "fs_op_ForgetInode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("ForgetInode"),
		},
		{
			name:      "fs_op_GetInodeAttributes",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("GetInodeAttributes"),
		},
		{
			name:      "fs_op_GetXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("GetXattr"),
		},
		{
			name:      "fs_op_ListXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("ListXattr"),
		},
		{
			name:      "fs_op_LookUpInode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("LookUpInode"),
		},
		{
			name:      "fs_op_MkDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("MkDir"),
		},
		{
			name:      "fs_op_MkNode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("MkNode"),
		},
		{
			name:      "fs_op_OpenDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("OpenDir"),
		},
		{
			name:      "fs_op_OpenFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("OpenFile"),
		},
		{
			name:      "fs_op_ReadDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("ReadDir"),
		},
		{
			name:      "fs_op_ReadDirPlus",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("ReadDirPlus"),
		},
		{
			name:      "fs_op_ReadFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("ReadFile"),
		},
		{
			name:      "fs_op_ReadSymlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("ReadSymlink"),
		},
		{
			name:      "fs_op_ReleaseDirHandle",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("ReleaseDirHandle"),
		},
		{
			name:      "fs_op_ReleaseFileHandle",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("ReleaseFileHandle"),
		},
		{
			name:      "fs_op_RemoveXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("RemoveXattr"),
		},
		{
			name:      "fs_op_Rename",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("Rename"),
		},
		{
			name:      "fs_op_RmDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("RmDir"),
		},
		{
			name:      "fs_op_SetInodeAttributes",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("SetInodeAttributes"),
		},
		{
			name:      "fs_op_SetXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("SetXattr"),
		},
		{
			name:      "fs_op_StatFS",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("StatFS"),
		},
		{
			name:      "fs_op_SyncFS",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("SyncFS"),
		},
		{
			name:      "fs_op_SyncFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("SyncFile"),
		},
		{
			name:      "fs_op_Unlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("Unlink"),
		},
		{
			name:      "fs_op_WriteFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOp("WriteFile"),
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
			name: "read_type_Parallel",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, ReadType("Parallel"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, ReadType("Random"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, ReadType("Sequential"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "read_type_Unknown",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, ReadType("Unknown"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Unknown")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, ReadType("Parallel"))
				m.GcsDownloadBytesCount(2, ReadType("Random"))
				m.GcsDownloadBytesCount(3, ReadType("Parallel"))
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(-5, ReadType("Parallel"))
				m.GcsDownloadBytesCount(2, ReadType("Parallel"))
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
				m.GcsReadCount(5, ReadType("Parallel"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, ReadType("Random"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, ReadType("Sequential"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "read_type_Unknown",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, ReadType("Unknown"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Unknown")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, ReadType("Parallel"))
				m.GcsReadCount(2, ReadType("Random"))
				m.GcsReadCount(3, ReadType("Parallel"))
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsReadCount(-5, ReadType("Parallel"))
				m.GcsReadCount(2, ReadType("Parallel"))
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
				m.GcsReaderCount(5, IoMethod("ReadHandle"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "ReadHandle")): 5,
			},
		},
		{
			name: "io_method_closed",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, IoMethod("closed"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "closed")): 5,
			},
		},
		{
			name: "io_method_opened",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, IoMethod("opened"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "opened")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, IoMethod("ReadHandle"))
				m.GcsReaderCount(2, IoMethod("closed"))
				m.GcsReaderCount(3, IoMethod("ReadHandle"))
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("io_method", "ReadHandle")): 8,
				attribute.NewSet(attribute.String("io_method", "closed")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(-5, IoMethod("ReadHandle"))
				m.GcsReaderCount(2, IoMethod("ReadHandle"))
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
				m.GcsRequestCount(5, GcsMethod("ComposeObjects"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "ComposeObjects")): 5,
			},
		},
		{
			name: "gcs_method_CopyObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("CopyObject"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CopyObject")): 5,
			},
		},
		{
			name: "gcs_method_CreateAppendableObjectWriter",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("CreateAppendableObjectWriter"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateAppendableObjectWriter")): 5,
			},
		},
		{
			name: "gcs_method_CreateFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("CreateFolder"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateFolder")): 5,
			},
		},
		{
			name: "gcs_method_CreateObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("CreateObject"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateObject")): 5,
			},
		},
		{
			name: "gcs_method_CreateObjectChunkWriter",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("CreateObjectChunkWriter"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateObjectChunkWriter")): 5,
			},
		},
		{
			name: "gcs_method_DeleteFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("DeleteFolder"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "DeleteFolder")): 5,
			},
		},
		{
			name: "gcs_method_DeleteObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("DeleteObject"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "DeleteObject")): 5,
			},
		},
		{
			name: "gcs_method_FinalizeUpload",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("FinalizeUpload"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "FinalizeUpload")): 5,
			},
		},
		{
			name: "gcs_method_FlushPendingWrites",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("FlushPendingWrites"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "FlushPendingWrites")): 5,
			},
		},
		{
			name: "gcs_method_GetFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("GetFolder"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "GetFolder")): 5,
			},
		},
		{
			name: "gcs_method_ListObjects",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("ListObjects"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "ListObjects")): 5,
			},
		},
		{
			name: "gcs_method_MoveObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("MoveObject"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "MoveObject")): 5,
			},
		},
		{
			name: "gcs_method_MultiRangeDownloader::Add",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("MultiRangeDownloader::Add"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "MultiRangeDownloader::Add")): 5,
			},
		},
		{
			name: "gcs_method_NewMultiRangeDownloader",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("NewMultiRangeDownloader"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "NewMultiRangeDownloader")): 5,
			},
		},
		{
			name: "gcs_method_NewReader",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("NewReader"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "NewReader")): 5,
			},
		},
		{
			name: "gcs_method_RenameFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("RenameFolder"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "RenameFolder")): 5,
			},
		},
		{
			name: "gcs_method_StatObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("StatObject"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "StatObject")): 5,
			},
		},
		{
			name: "gcs_method_UpdateObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("UpdateObject"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "UpdateObject")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethod("ComposeObjects"))
				m.GcsRequestCount(2, GcsMethod("CopyObject"))
				m.GcsRequestCount(3, GcsMethod("ComposeObjects"))
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("gcs_method", "ComposeObjects")): 8,
				attribute.NewSet(attribute.String("gcs_method", "CopyObject")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(-5, GcsMethod("ComposeObjects"))
				m.GcsRequestCount(2, GcsMethod("ComposeObjects"))
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
			gcsMethod: GcsMethod("ComposeObjects"),
		},
		{
			name:      "gcs_method_CopyObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("CopyObject"),
		},
		{
			name:      "gcs_method_CreateAppendableObjectWriter",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("CreateAppendableObjectWriter"),
		},
		{
			name:      "gcs_method_CreateFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("CreateFolder"),
		},
		{
			name:      "gcs_method_CreateObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("CreateObject"),
		},
		{
			name:      "gcs_method_CreateObjectChunkWriter",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("CreateObjectChunkWriter"),
		},
		{
			name:      "gcs_method_DeleteFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("DeleteFolder"),
		},
		{
			name:      "gcs_method_DeleteObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("DeleteObject"),
		},
		{
			name:      "gcs_method_FinalizeUpload",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("FinalizeUpload"),
		},
		{
			name:      "gcs_method_FlushPendingWrites",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("FlushPendingWrites"),
		},
		{
			name:      "gcs_method_GetFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("GetFolder"),
		},
		{
			name:      "gcs_method_ListObjects",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("ListObjects"),
		},
		{
			name:      "gcs_method_MoveObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("MoveObject"),
		},
		{
			name:      "gcs_method_MultiRangeDownloader::Add",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("MultiRangeDownloader::Add"),
		},
		{
			name:      "gcs_method_NewMultiRangeDownloader",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("NewMultiRangeDownloader"),
		},
		{
			name:      "gcs_method_NewReader",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("NewReader"),
		},
		{
			name:      "gcs_method_RenameFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("RenameFolder"),
		},
		{
			name:      "gcs_method_StatObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("StatObject"),
		},
		{
			name:      "gcs_method_UpdateObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethod("UpdateObject"),
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
				m.GcsRetryCount(5, RetryErrorCategory("OTHER_ERRORS"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", "OTHER_ERRORS")): 5,
			},
		},
		{
			name: "retry_error_category_STALLED_READ_REQUEST",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(5, RetryErrorCategory("STALLED_READ_REQUEST"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", "STALLED_READ_REQUEST")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(5, RetryErrorCategory("OTHER_ERRORS"))
				m.GcsRetryCount(2, RetryErrorCategory("STALLED_READ_REQUEST"))
				m.GcsRetryCount(3, RetryErrorCategory("OTHER_ERRORS"))
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("retry_error_category", "OTHER_ERRORS")): 8,
				attribute.NewSet(attribute.String("retry_error_category", "STALLED_READ_REQUEST")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(-5, RetryErrorCategory("OTHER_ERRORS"))
				m.GcsRetryCount(2, RetryErrorCategory("OTHER_ERRORS"))
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
				m.TestUpdownCounterWithAttrs(5, RequestType("attr1"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("request_type", "attr1")): 5,
			},
		},
		{
			name: "request_type_attr2",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(5, RequestType("attr2"))
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("request_type", "attr2")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(5, RequestType("attr1"))
				m.TestUpdownCounterWithAttrs(2, RequestType("attr2"))
				m.TestUpdownCounterWithAttrs(3, RequestType("attr1"))
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("request_type", "attr1")): 8,
				attribute.NewSet(attribute.String("request_type", "attr2")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(-5, RequestType("attr1"))
				m.TestUpdownCounterWithAttrs(2, RequestType("attr1"))
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
