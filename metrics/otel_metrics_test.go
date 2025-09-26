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
			status:    StatusCancelledAttr,
		},
		{
			name:      "status_failed",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			status:    StatusFailedAttr,
		},
		{
			name:      "status_successful",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			status:    StatusSuccessfulAttr,
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
				m.BufferedReadFallbackTriggerCount(5, ReasonInsufficientMemoryAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("reason", "insufficient_memory")): 5,
			},
		},
		{
			name: "reason_random_read_detected",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(5, ReasonRandomReadDetectedAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("reason", "random_read_detected")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(5, ReasonInsufficientMemoryAttr)
				m.BufferedReadFallbackTriggerCount(2, ReasonRandomReadDetectedAttr)
				m.BufferedReadFallbackTriggerCount(3, ReasonInsufficientMemoryAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("reason", "insufficient_memory")): 8,
				attribute.NewSet(attribute.String("reason", "random_read_detected")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(-5, ReasonInsufficientMemoryAttr)
				m.BufferedReadFallbackTriggerCount(2, ReasonInsufficientMemoryAttr)
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
				m.BufferedReadScheduledBlockCount(5, StatusCancelledAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("status", "cancelled")): 5,
			},
		},
		{
			name: "status_failed",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(5, StatusFailedAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("status", "failed")): 5,
			},
		},
		{
			name: "status_successful",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(5, StatusSuccessfulAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("status", "successful")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(5, StatusCancelledAttr)
				m.BufferedReadScheduledBlockCount(2, StatusFailedAttr)
				m.BufferedReadScheduledBlockCount(3, StatusCancelledAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("status", "cancelled")): 8,
				attribute.NewSet(attribute.String("status", "failed")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(-5, StatusCancelledAttr)
				m.BufferedReadScheduledBlockCount(2, StatusCancelledAttr)
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
				m.FileCacheReadBytesCount(5, ReadTypeParallelAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, ReadTypeRandomAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, ReadTypeSequentialAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, ReadTypeParallelAttr)
				m.FileCacheReadBytesCount(2, ReadTypeRandomAttr)
				m.FileCacheReadBytesCount(3, ReadTypeParallelAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(-5, ReadTypeParallelAttr)
				m.FileCacheReadBytesCount(2, ReadTypeParallelAttr)
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
				m.FileCacheReadCount(5, true, ReadTypeParallelAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "cache_hit_true_read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, ReadTypeRandomAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "cache_hit_true_read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, ReadTypeSequentialAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Parallel",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, ReadTypeParallelAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, ReadTypeRandomAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, ReadTypeSequentialAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Sequential")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, ReadTypeParallelAttr)
				m.FileCacheReadCount(2, true, ReadTypeRandomAttr)
				m.FileCacheReadCount(3, true, ReadTypeParallelAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(-5, true, ReadTypeParallelAttr)
				m.FileCacheReadCount(2, true, ReadTypeParallelAttr)
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
				m.FsOpsCount(5, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "WriteFile")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpBatchForgetAttr)
				m.FsOpsCount(2, FsOpCreateFileAttr)
				m.FsOpsCount(3, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_op", "BatchForget")): 8,
				attribute.NewSet(attribute.String("fs_op", "CreateFile")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FsOpsCount(-5, FsOpBatchForgetAttr)
				m.FsOpsCount(2, FsOpBatchForgetAttr)
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
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTYAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERRORAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTSAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERRORAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENTAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATIONAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERRORAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERRORAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERRORAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIRAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTEDAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIRAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERRORAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERRORAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpCreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpCreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpCreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpFallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpFlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpGetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpGetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpLookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpMkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpMkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpOpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpOpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpRemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpRenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpRmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpSetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpSetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpStatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpSyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpSyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpUnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILESAttr, FsOpWriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "WriteFile")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERRORAttr, FsOpBatchForgetAttr)
				m.FsOpsErrorCount(2, FsErrorCategoryDEVICEERRORAttr, FsOpCreateFileAttr)
				m.FsOpsErrorCount(3, FsErrorCategoryDEVICEERRORAttr, FsOpBatchForgetAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "BatchForget")): 8,
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateFile")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(-5, FsErrorCategoryDEVICEERRORAttr, FsOpBatchForgetAttr)
				m.FsOpsErrorCount(2, FsErrorCategoryDEVICEERRORAttr, FsOpBatchForgetAttr)
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
			fsOp:      FsOpBatchForgetAttr,
		},
		{
			name:      "fs_op_CreateFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpCreateFileAttr,
		},
		{
			name:      "fs_op_CreateLink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpCreateLinkAttr,
		},
		{
			name:      "fs_op_CreateSymlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpCreateSymlinkAttr,
		},
		{
			name:      "fs_op_Fallocate",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpFallocateAttr,
		},
		{
			name:      "fs_op_FlushFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpFlushFileAttr,
		},
		{
			name:      "fs_op_ForgetInode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpForgetInodeAttr,
		},
		{
			name:      "fs_op_GetInodeAttributes",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpGetInodeAttributesAttr,
		},
		{
			name:      "fs_op_GetXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpGetXattrAttr,
		},
		{
			name:      "fs_op_ListXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpListXattrAttr,
		},
		{
			name:      "fs_op_LookUpInode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpLookUpInodeAttr,
		},
		{
			name:      "fs_op_MkDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpMkDirAttr,
		},
		{
			name:      "fs_op_MkNode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpMkNodeAttr,
		},
		{
			name:      "fs_op_OpenDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpOpenDirAttr,
		},
		{
			name:      "fs_op_OpenFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpOpenFileAttr,
		},
		{
			name:      "fs_op_ReadDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpReadDirAttr,
		},
		{
			name:      "fs_op_ReadDirPlus",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpReadDirPlusAttr,
		},
		{
			name:      "fs_op_ReadFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpReadFileAttr,
		},
		{
			name:      "fs_op_ReadSymlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpReadSymlinkAttr,
		},
		{
			name:      "fs_op_ReleaseDirHandle",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpReleaseDirHandleAttr,
		},
		{
			name:      "fs_op_ReleaseFileHandle",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpReleaseFileHandleAttr,
		},
		{
			name:      "fs_op_RemoveXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpRemoveXattrAttr,
		},
		{
			name:      "fs_op_Rename",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpRenameAttr,
		},
		{
			name:      "fs_op_RmDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpRmDirAttr,
		},
		{
			name:      "fs_op_SetInodeAttributes",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpSetInodeAttributesAttr,
		},
		{
			name:      "fs_op_SetXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpSetXattrAttr,
		},
		{
			name:      "fs_op_StatFS",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpStatFSAttr,
		},
		{
			name:      "fs_op_SyncFS",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpSyncFSAttr,
		},
		{
			name:      "fs_op_SyncFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpSyncFileAttr,
		},
		{
			name:      "fs_op_Unlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpUnlinkAttr,
		},
		{
			name:      "fs_op_WriteFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpWriteFileAttr,
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
				m.GcsDownloadBytesCount(5, ReadTypeParallelAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, ReadTypeRandomAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, ReadTypeSequentialAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, ReadTypeParallelAttr)
				m.GcsDownloadBytesCount(2, ReadTypeRandomAttr)
				m.GcsDownloadBytesCount(3, ReadTypeParallelAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(-5, ReadTypeParallelAttr)
				m.GcsDownloadBytesCount(2, ReadTypeParallelAttr)
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
				m.GcsReadCount(5, ReadTypeParallelAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, ReadTypeRandomAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, ReadTypeSequentialAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, ReadTypeParallelAttr)
				m.GcsReadCount(2, ReadTypeRandomAttr)
				m.GcsReadCount(3, ReadTypeParallelAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsReadCount(-5, ReadTypeParallelAttr)
				m.GcsReadCount(2, ReadTypeParallelAttr)
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
				m.GcsReaderCount(5, IoMethodReadHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "ReadHandle")): 5,
			},
		},
		{
			name: "io_method_closed",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, IoMethodClosedAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "closed")): 5,
			},
		},
		{
			name: "io_method_opened",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, IoMethodOpenedAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "opened")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, IoMethodReadHandleAttr)
				m.GcsReaderCount(2, IoMethodClosedAttr)
				m.GcsReaderCount(3, IoMethodReadHandleAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("io_method", "ReadHandle")): 8,
				attribute.NewSet(attribute.String("io_method", "closed")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(-5, IoMethodReadHandleAttr)
				m.GcsReaderCount(2, IoMethodReadHandleAttr)
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
				m.GcsRequestCount(5, GcsMethodComposeObjectsAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "ComposeObjects")): 5,
			},
		},
		{
			name: "gcs_method_CopyObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodCopyObjectAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CopyObject")): 5,
			},
		},
		{
			name: "gcs_method_CreateAppendableObjectWriter",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodCreateAppendableObjectWriterAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateAppendableObjectWriter")): 5,
			},
		},
		{
			name: "gcs_method_CreateFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodCreateFolderAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateFolder")): 5,
			},
		},
		{
			name: "gcs_method_CreateObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodCreateObjectAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateObject")): 5,
			},
		},
		{
			name: "gcs_method_CreateObjectChunkWriter",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodCreateObjectChunkWriterAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateObjectChunkWriter")): 5,
			},
		},
		{
			name: "gcs_method_DeleteFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodDeleteFolderAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "DeleteFolder")): 5,
			},
		},
		{
			name: "gcs_method_DeleteObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodDeleteObjectAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "DeleteObject")): 5,
			},
		},
		{
			name: "gcs_method_FinalizeUpload",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodFinalizeUploadAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "FinalizeUpload")): 5,
			},
		},
		{
			name: "gcs_method_FlushPendingWrites",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodFlushPendingWritesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "FlushPendingWrites")): 5,
			},
		},
		{
			name: "gcs_method_GetFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodGetFolderAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "GetFolder")): 5,
			},
		},
		{
			name: "gcs_method_ListObjects",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodListObjectsAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "ListObjects")): 5,
			},
		},
		{
			name: "gcs_method_MoveObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodMoveObjectAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "MoveObject")): 5,
			},
		},
		{
			name: "gcs_method_MultiRangeDownloader::Add",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodMultiRangeDownloaderAddAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "MultiRangeDownloader::Add")): 5,
			},
		},
		{
			name: "gcs_method_NewMultiRangeDownloader",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodNewMultiRangeDownloaderAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "NewMultiRangeDownloader")): 5,
			},
		},
		{
			name: "gcs_method_NewReader",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodNewReaderAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "NewReader")): 5,
			},
		},
		{
			name: "gcs_method_RenameFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodRenameFolderAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "RenameFolder")): 5,
			},
		},
		{
			name: "gcs_method_StatObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodStatObjectAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "StatObject")): 5,
			},
		},
		{
			name: "gcs_method_UpdateObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodUpdateObjectAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "UpdateObject")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodComposeObjectsAttr)
				m.GcsRequestCount(2, GcsMethodCopyObjectAttr)
				m.GcsRequestCount(3, GcsMethodComposeObjectsAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("gcs_method", "ComposeObjects")): 8,
				attribute.NewSet(attribute.String("gcs_method", "CopyObject")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(-5, GcsMethodComposeObjectsAttr)
				m.GcsRequestCount(2, GcsMethodComposeObjectsAttr)
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
			gcsMethod: GcsMethodComposeObjectsAttr,
		},
		{
			name:      "gcs_method_CopyObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodCopyObjectAttr,
		},
		{
			name:      "gcs_method_CreateAppendableObjectWriter",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodCreateAppendableObjectWriterAttr,
		},
		{
			name:      "gcs_method_CreateFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodCreateFolderAttr,
		},
		{
			name:      "gcs_method_CreateObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodCreateObjectAttr,
		},
		{
			name:      "gcs_method_CreateObjectChunkWriter",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodCreateObjectChunkWriterAttr,
		},
		{
			name:      "gcs_method_DeleteFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodDeleteFolderAttr,
		},
		{
			name:      "gcs_method_DeleteObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodDeleteObjectAttr,
		},
		{
			name:      "gcs_method_FinalizeUpload",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodFinalizeUploadAttr,
		},
		{
			name:      "gcs_method_FlushPendingWrites",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodFlushPendingWritesAttr,
		},
		{
			name:      "gcs_method_GetFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodGetFolderAttr,
		},
		{
			name:      "gcs_method_ListObjects",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodListObjectsAttr,
		},
		{
			name:      "gcs_method_MoveObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodMoveObjectAttr,
		},
		{
			name:      "gcs_method_MultiRangeDownloader::Add",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodMultiRangeDownloaderAddAttr,
		},
		{
			name:      "gcs_method_NewMultiRangeDownloader",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodNewMultiRangeDownloaderAttr,
		},
		{
			name:      "gcs_method_NewReader",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodNewReaderAttr,
		},
		{
			name:      "gcs_method_RenameFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodRenameFolderAttr,
		},
		{
			name:      "gcs_method_StatObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodStatObjectAttr,
		},
		{
			name:      "gcs_method_UpdateObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodUpdateObjectAttr,
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
				m.GcsRetryCount(5, RetryErrorCategoryOTHERERRORSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", "OTHER_ERRORS")): 5,
			},
		},
		{
			name: "retry_error_category_STALLED_READ_REQUEST",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(5, RetryErrorCategorySTALLEDREADREQUESTAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", "STALLED_READ_REQUEST")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(5, RetryErrorCategoryOTHERERRORSAttr)
				m.GcsRetryCount(2, RetryErrorCategorySTALLEDREADREQUESTAttr)
				m.GcsRetryCount(3, RetryErrorCategoryOTHERERRORSAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("retry_error_category", "OTHER_ERRORS")): 8,
				attribute.NewSet(attribute.String("retry_error_category", "STALLED_READ_REQUEST")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(-5, RetryErrorCategoryOTHERERRORSAttr)
				m.GcsRetryCount(2, RetryErrorCategoryOTHERERRORSAttr)
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
				m.TestUpdownCounterWithAttrs(5, RequestTypeAttr1Attr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("request_type", "attr1")): 5,
			},
		},
		{
			name: "request_type_attr2",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(5, RequestTypeAttr2Attr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("request_type", "attr2")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(5, RequestTypeAttr1Attr)
				m.TestUpdownCounterWithAttrs(2, RequestTypeAttr2Attr)
				m.TestUpdownCounterWithAttrs(3, RequestTypeAttr1Attr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("request_type", "attr1")): 8,
				attribute.NewSet(attribute.String("request_type", "attr2")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(-5, RequestTypeAttr1Attr)
				m.TestUpdownCounterWithAttrs(2, RequestTypeAttr1Attr)
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
