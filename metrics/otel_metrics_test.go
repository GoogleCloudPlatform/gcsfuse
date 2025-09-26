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
			status:    StatusCancelled,
		},
		{
			name:      "status_failed",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			status:    StatusFailed,
		},
		{
			name:      "status_successful",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			status:    StatusSuccessful,
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
				m.BufferedReadFallbackTriggerCount(5, ReasonInsufficientMemory)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("reason", "insufficient_memory")): 5,
			},
		},
		{
			name: "reason_random_read_detected",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(5, ReasonRandomReadDetected)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("reason", "random_read_detected")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(5, ReasonInsufficientMemory)
				m.BufferedReadFallbackTriggerCount(2, ReasonRandomReadDetected)
				m.BufferedReadFallbackTriggerCount(3, ReasonInsufficientMemory)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("reason", "insufficient_memory")): 8,
				attribute.NewSet(attribute.String("reason", "random_read_detected")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(-5, ReasonInsufficientMemory)
				m.BufferedReadFallbackTriggerCount(2, ReasonInsufficientMemory)
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
				m.BufferedReadScheduledBlockCount(5, StatusCancelled)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("status", "cancelled")): 5,
			},
		},
		{
			name: "status_failed",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(5, StatusFailed)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("status", "failed")): 5,
			},
		},
		{
			name: "status_successful",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(5, StatusSuccessful)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("status", "successful")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(5, StatusCancelled)
				m.BufferedReadScheduledBlockCount(2, StatusFailed)
				m.BufferedReadScheduledBlockCount(3, StatusCancelled)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("status", "cancelled")): 8,
				attribute.NewSet(attribute.String("status", "failed")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(-5, StatusCancelled)
				m.BufferedReadScheduledBlockCount(2, StatusCancelled)
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
				m.FileCacheReadBytesCount(5, ReadTypeParallel)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, ReadTypeRandom)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, ReadTypeSequential)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, ReadTypeParallel)
				m.FileCacheReadBytesCount(2, ReadTypeRandom)
				m.FileCacheReadBytesCount(3, ReadTypeParallel)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(-5, ReadTypeParallel)
				m.FileCacheReadBytesCount(2, ReadTypeParallel)
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
				m.FileCacheReadCount(5, true, ReadTypeParallel)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "cache_hit_true_read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, ReadTypeRandom)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "cache_hit_true_read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, ReadTypeSequential)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Parallel",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, ReadTypeParallel)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, ReadTypeRandom)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, ReadTypeSequential)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Sequential")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, ReadTypeParallel)
				m.FileCacheReadCount(2, true, ReadTypeRandom)
				m.FileCacheReadCount(3, true, ReadTypeParallel)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(-5, true, ReadTypeParallel)
				m.FileCacheReadCount(2, true, ReadTypeParallel)
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
				m.FsOpsCount(5, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "WriteFile")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FsOpBatchForget)
				m.FsOpsCount(2, FsOpCreateFile)
				m.FsOpsCount(3, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_op", "BatchForget")): 8,
				attribute.NewSet(attribute.String("fs_op", "CreateFile")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FsOpsCount(-5, FsOpBatchForget)
				m.FsOpsCount(2, FsOpBatchForget)
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
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDIRNOTEMPTY, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "DIR_NOT_EMPTY"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEDIRERROR, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_DIR_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryFILEEXISTS, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "FILE_EXISTS"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINTERRUPTERROR, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INTERRUPT_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDARGUMENT, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_ARGUMENT"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryINVALIDOPERATION, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "INVALID_OPERATION"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryIOERROR, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "IO_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryMISCERROR, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "MISC_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNETWORKERROR, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NETWORK_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTADIR, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_A_DIR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOTIMPLEMENTED, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NOT_IMPLEMENTED"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryNOFILEORDIR, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "NO_FILE_OR_DIR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPERMERROR, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PERM_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryPROCESSRESOURCEMGMTERROR, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "PROCESS_RESOURCE_MGMT_ERROR"), attribute.String("fs_op", "WriteFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "BatchForget")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpCreateFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpCreateLink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateLink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpCreateSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "CreateSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpFallocate)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Fallocate")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpFlushFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "FlushFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpForgetInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ForgetInode")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpGetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "GetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpGetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "GetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpListXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ListXattr")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpLookUpInode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "LookUpInode")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpMkDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "MkDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpMkNode)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "MkNode")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpOpenDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "OpenDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpOpenFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "OpenFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpReadDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpReadDirPlus)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadDirPlus")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpReadFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpReadSymlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReadSymlink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpReleaseDirHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReleaseDirHandle")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpReleaseFileHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "ReleaseFileHandle")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpRemoveXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "RemoveXattr")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpRename)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Rename")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpRmDir)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "RmDir")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpSetInodeAttributes)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SetInodeAttributes")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpSetXattr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SetXattr")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpStatFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "StatFS")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpSyncFS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SyncFS")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpSyncFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "SyncFile")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpUnlink)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "Unlink")): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryTOOMANYOPENFILES, FsOpWriteFile)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", "TOO_MANY_OPEN_FILES"), attribute.String("fs_op", "WriteFile")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FsErrorCategoryDEVICEERROR, FsOpBatchForget)
				m.FsOpsErrorCount(2, FsErrorCategoryDEVICEERROR, FsOpCreateFile)
				m.FsOpsErrorCount(3, FsErrorCategoryDEVICEERROR, FsOpBatchForget)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "BatchForget")): 8,
				attribute.NewSet(attribute.String("fs_error_category", "DEVICE_ERROR"), attribute.String("fs_op", "CreateFile")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(-5, FsErrorCategoryDEVICEERROR, FsOpBatchForget)
				m.FsOpsErrorCount(2, FsErrorCategoryDEVICEERROR, FsOpBatchForget)
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
			fsOp:      FsOpBatchForget,
		},
		{
			name:      "fs_op_CreateFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpCreateFile,
		},
		{
			name:      "fs_op_CreateLink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpCreateLink,
		},
		{
			name:      "fs_op_CreateSymlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpCreateSymlink,
		},
		{
			name:      "fs_op_Fallocate",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpFallocate,
		},
		{
			name:      "fs_op_FlushFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpFlushFile,
		},
		{
			name:      "fs_op_ForgetInode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpForgetInode,
		},
		{
			name:      "fs_op_GetInodeAttributes",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpGetInodeAttributes,
		},
		{
			name:      "fs_op_GetXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpGetXattr,
		},
		{
			name:      "fs_op_ListXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpListXattr,
		},
		{
			name:      "fs_op_LookUpInode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpLookUpInode,
		},
		{
			name:      "fs_op_MkDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpMkDir,
		},
		{
			name:      "fs_op_MkNode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpMkNode,
		},
		{
			name:      "fs_op_OpenDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpOpenDir,
		},
		{
			name:      "fs_op_OpenFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpOpenFile,
		},
		{
			name:      "fs_op_ReadDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpReadDir,
		},
		{
			name:      "fs_op_ReadDirPlus",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpReadDirPlus,
		},
		{
			name:      "fs_op_ReadFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpReadFile,
		},
		{
			name:      "fs_op_ReadSymlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpReadSymlink,
		},
		{
			name:      "fs_op_ReleaseDirHandle",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpReleaseDirHandle,
		},
		{
			name:      "fs_op_ReleaseFileHandle",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpReleaseFileHandle,
		},
		{
			name:      "fs_op_RemoveXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpRemoveXattr,
		},
		{
			name:      "fs_op_Rename",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpRename,
		},
		{
			name:      "fs_op_RmDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpRmDir,
		},
		{
			name:      "fs_op_SetInodeAttributes",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpSetInodeAttributes,
		},
		{
			name:      "fs_op_SetXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpSetXattr,
		},
		{
			name:      "fs_op_StatFS",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpStatFS,
		},
		{
			name:      "fs_op_SyncFS",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpSyncFS,
		},
		{
			name:      "fs_op_SyncFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpSyncFile,
		},
		{
			name:      "fs_op_Unlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpUnlink,
		},
		{
			name:      "fs_op_WriteFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FsOpWriteFile,
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
				m.GcsDownloadBytesCount(5, ReadTypeParallel)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, ReadTypeRandom)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, ReadTypeSequential)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, ReadTypeParallel)
				m.GcsDownloadBytesCount(2, ReadTypeRandom)
				m.GcsDownloadBytesCount(3, ReadTypeParallel)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(-5, ReadTypeParallel)
				m.GcsDownloadBytesCount(2, ReadTypeParallel)
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
				m.GcsReadCount(5, ReadTypeParallel)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, ReadTypeRandom)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, ReadTypeSequential)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, ReadTypeParallel)
				m.GcsReadCount(2, ReadTypeRandom)
				m.GcsReadCount(3, ReadTypeParallel)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", "Parallel")): 8,
				attribute.NewSet(attribute.String("read_type", "Random")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsReadCount(-5, ReadTypeParallel)
				m.GcsReadCount(2, ReadTypeParallel)
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
				m.GcsReaderCount(5, IoMethodReadHandle)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "ReadHandle")): 5,
			},
		},
		{
			name: "io_method_closed",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, IoMethodClosed)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "closed")): 5,
			},
		},
		{
			name: "io_method_opened",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, IoMethodOpened)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "opened")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, IoMethodReadHandle)
				m.GcsReaderCount(2, IoMethodClosed)
				m.GcsReaderCount(3, IoMethodReadHandle)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("io_method", "ReadHandle")): 8,
				attribute.NewSet(attribute.String("io_method", "closed")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(-5, IoMethodReadHandle)
				m.GcsReaderCount(2, IoMethodReadHandle)
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
				m.GcsRequestCount(5, GcsMethodComposeObjects)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "ComposeObjects")): 5,
			},
		},
		{
			name: "gcs_method_CopyObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodCopyObject)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CopyObject")): 5,
			},
		},
		{
			name: "gcs_method_CreateAppendableObjectWriter",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodCreateAppendableObjectWriter)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateAppendableObjectWriter")): 5,
			},
		},
		{
			name: "gcs_method_CreateFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodCreateFolder)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateFolder")): 5,
			},
		},
		{
			name: "gcs_method_CreateObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodCreateObject)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateObject")): 5,
			},
		},
		{
			name: "gcs_method_CreateObjectChunkWriter",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodCreateObjectChunkWriter)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "CreateObjectChunkWriter")): 5,
			},
		},
		{
			name: "gcs_method_DeleteFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodDeleteFolder)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "DeleteFolder")): 5,
			},
		},
		{
			name: "gcs_method_DeleteObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodDeleteObject)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "DeleteObject")): 5,
			},
		},
		{
			name: "gcs_method_FinalizeUpload",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodFinalizeUpload)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "FinalizeUpload")): 5,
			},
		},
		{
			name: "gcs_method_FlushPendingWrites",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodFlushPendingWrites)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "FlushPendingWrites")): 5,
			},
		},
		{
			name: "gcs_method_GetFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodGetFolder)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "GetFolder")): 5,
			},
		},
		{
			name: "gcs_method_ListObjects",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodListObjects)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "ListObjects")): 5,
			},
		},
		{
			name: "gcs_method_MoveObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodMoveObject)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "MoveObject")): 5,
			},
		},
		{
			name: "gcs_method_MultiRangeDownloader::Add",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodMultiRangeDownloaderAdd)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "MultiRangeDownloader::Add")): 5,
			},
		},
		{
			name: "gcs_method_NewMultiRangeDownloader",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodNewMultiRangeDownloader)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "NewMultiRangeDownloader")): 5,
			},
		},
		{
			name: "gcs_method_NewReader",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodNewReader)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "NewReader")): 5,
			},
		},
		{
			name: "gcs_method_RenameFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodRenameFolder)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "RenameFolder")): 5,
			},
		},
		{
			name: "gcs_method_StatObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodStatObject)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "StatObject")): 5,
			},
		},
		{
			name: "gcs_method_UpdateObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodUpdateObject)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", "UpdateObject")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GcsMethodComposeObjects)
				m.GcsRequestCount(2, GcsMethodCopyObject)
				m.GcsRequestCount(3, GcsMethodComposeObjects)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("gcs_method", "ComposeObjects")): 8,
				attribute.NewSet(attribute.String("gcs_method", "CopyObject")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(-5, GcsMethodComposeObjects)
				m.GcsRequestCount(2, GcsMethodComposeObjects)
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
			gcsMethod: GcsMethodComposeObjects,
		},
		{
			name:      "gcs_method_CopyObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodCopyObject,
		},
		{
			name:      "gcs_method_CreateAppendableObjectWriter",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodCreateAppendableObjectWriter,
		},
		{
			name:      "gcs_method_CreateFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodCreateFolder,
		},
		{
			name:      "gcs_method_CreateObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodCreateObject,
		},
		{
			name:      "gcs_method_CreateObjectChunkWriter",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodCreateObjectChunkWriter,
		},
		{
			name:      "gcs_method_DeleteFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodDeleteFolder,
		},
		{
			name:      "gcs_method_DeleteObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodDeleteObject,
		},
		{
			name:      "gcs_method_FinalizeUpload",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodFinalizeUpload,
		},
		{
			name:      "gcs_method_FlushPendingWrites",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodFlushPendingWrites,
		},
		{
			name:      "gcs_method_GetFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodGetFolder,
		},
		{
			name:      "gcs_method_ListObjects",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodListObjects,
		},
		{
			name:      "gcs_method_MoveObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodMoveObject,
		},
		{
			name:      "gcs_method_MultiRangeDownloader::Add",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodMultiRangeDownloaderAdd,
		},
		{
			name:      "gcs_method_NewMultiRangeDownloader",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodNewMultiRangeDownloader,
		},
		{
			name:      "gcs_method_NewReader",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodNewReader,
		},
		{
			name:      "gcs_method_RenameFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodRenameFolder,
		},
		{
			name:      "gcs_method_StatObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodStatObject,
		},
		{
			name:      "gcs_method_UpdateObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GcsMethodUpdateObject,
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
				m.GcsRetryCount(5, RetryErrorCategoryOTHERERRORS)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", "OTHER_ERRORS")): 5,
			},
		},
		{
			name: "retry_error_category_STALLED_READ_REQUEST",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(5, RetryErrorCategorySTALLEDREADREQUEST)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", "STALLED_READ_REQUEST")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(5, RetryErrorCategoryOTHERERRORS)
				m.GcsRetryCount(2, RetryErrorCategorySTALLEDREADREQUEST)
				m.GcsRetryCount(3, RetryErrorCategoryOTHERERRORS)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("retry_error_category", "OTHER_ERRORS")): 8,
				attribute.NewSet(attribute.String("retry_error_category", "STALLED_READ_REQUEST")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(-5, RetryErrorCategoryOTHERERRORS)
				m.GcsRetryCount(2, RetryErrorCategoryOTHERERRORS)
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
				m.TestUpdownCounterWithAttrs(5, RequestTypeAttr1)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("request_type", "attr1")): 5,
			},
		},
		{
			name: "request_type_attr2",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(5, RequestTypeAttr2)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("request_type", "attr2")): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(5, RequestTypeAttr1)
				m.TestUpdownCounterWithAttrs(2, RequestTypeAttr2)
				m.TestUpdownCounterWithAttrs(3, RequestTypeAttr1)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("request_type", "attr1")): 8,
				attribute.NewSet(attribute.String("request_type", "attr2")): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(-5, RequestTypeAttr1)
				m.TestUpdownCounterWithAttrs(2, RequestTypeAttr1)
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
