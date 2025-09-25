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
		status    MetricAttr
	}{
		{
			name:      "status_cancelled",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			status:    CancelledAttr,
		},
		{
			name:      "status_failed",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			status:    FailedAttr,
		},
		{
			name:      "status_successful",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			status:    SuccessfulAttr,
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
				m.BufferedReadFallbackTriggerCount(5, InsufficientMemoryAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("reason", string(InsufficientMemoryAttr))): 5,
			},
		},
		{
			name: "reason_random_read_detected",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(5, RandomReadDetectedAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("reason", string(RandomReadDetectedAttr))): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(5, InsufficientMemoryAttr)
				m.BufferedReadFallbackTriggerCount(2, RandomReadDetectedAttr)
				m.BufferedReadFallbackTriggerCount(3, InsufficientMemoryAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("reason", string(InsufficientMemoryAttr))): 8,
				attribute.NewSet(attribute.String("reason", string(RandomReadDetectedAttr))): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.BufferedReadFallbackTriggerCount(-5, InsufficientMemoryAttr)
				m.BufferedReadFallbackTriggerCount(2, InsufficientMemoryAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("reason", string(InsufficientMemoryAttr))): 2},
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
				m.BufferedReadScheduledBlockCount(5, CancelledAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("status", string(CancelledAttr))): 5,
			},
		},
		{
			name: "status_failed",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(5, FailedAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("status", string(FailedAttr))): 5,
			},
		},
		{
			name: "status_successful",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(5, SuccessfulAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("status", string(SuccessfulAttr))): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(5, CancelledAttr)
				m.BufferedReadScheduledBlockCount(2, FailedAttr)
				m.BufferedReadScheduledBlockCount(3, CancelledAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("status", string(CancelledAttr))): 8,
				attribute.NewSet(attribute.String("status", string(FailedAttr))): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.BufferedReadScheduledBlockCount(-5, CancelledAttr)
				m.BufferedReadScheduledBlockCount(2, CancelledAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("status", string(CancelledAttr))): 2},
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
				m.FileCacheReadBytesCount(5, ParallelAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", string(ParallelAttr))): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, RandomAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", string(RandomAttr))): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, SequentialAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", string(SequentialAttr))): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(5, ParallelAttr)
				m.FileCacheReadBytesCount(2, RandomAttr)
				m.FileCacheReadBytesCount(3, ParallelAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", string(ParallelAttr))): 8,
				attribute.NewSet(attribute.String("read_type", string(RandomAttr))): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(-5, ParallelAttr)
				m.FileCacheReadBytesCount(2, ParallelAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", string(ParallelAttr))): 2},
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
				m.FileCacheReadCount(5, true, ParallelAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", string(ParallelAttr))): 5,
			},
		},
		{
			name: "cache_hit_true_read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, RandomAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", string(RandomAttr))): 5,
			},
		},
		{
			name: "cache_hit_true_read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, SequentialAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", string(SequentialAttr))): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Parallel",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, ParallelAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", string(ParallelAttr))): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, RandomAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", string(RandomAttr))): 5,
			},
		},
		{
			name: "cache_hit_false_read_type_Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, SequentialAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", string(SequentialAttr))): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, ParallelAttr)
				m.FileCacheReadCount(2, true, RandomAttr)
				m.FileCacheReadCount(3, true, ParallelAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", string(ParallelAttr))): 8,
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", string(RandomAttr))): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(-5, true, ParallelAttr)
				m.FileCacheReadCount(2, true, ParallelAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", string(ParallelAttr))): 2},
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
				m.FsOpsCount(5, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, BatchForgetAttr)
				m.FsOpsCount(2, CreateFileAttr)
				m.FsOpsCount(3, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_op", string(BatchForgetAttr))): 8,
				attribute.NewSet(attribute.String("fs_op", string(CreateFileAttr))): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FsOpsCount(-5, BatchForgetAttr)
				m.FsOpsCount(2, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_op", string(BatchForgetAttr))): 2},
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
				m.FsOpsErrorCount(5, DEVICEERRORAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DEVICE_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_DIR_NOT_EMPTY_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DIRNOTEMPTYAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(DIRNOTEMPTYAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_DIR_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEDIRERRORAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEDIRERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_FILE_EXISTS_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, FILEEXISTSAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(FILEEXISTSAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INTERRUPT_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INTERRUPTERRORAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INTERRUPTERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_ARGUMENT_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDARGUMENTAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDARGUMENTAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_INVALID_OPERATION_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, INVALIDOPERATIONAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(INVALIDOPERATIONAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_IO_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, IOERRORAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(IOERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_MISC_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, MISCERRORAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(MISCERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NETWORK_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NETWORKERRORAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NETWORKERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_A_DIR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTADIRAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTADIRAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NOT_IMPLEMENTED_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOTIMPLEMENTEDAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOTIMPLEMENTEDAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_NO_FILE_OR_DIR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, NOFILEORDIRAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(NOFILEORDIRAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PERM_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PERMERRORAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PERMERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_PROCESS_RESOURCE_MGMT_ERROR_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, PROCESSRESOURCEMGMTERRORAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(PROCESSRESOURCEMGMTERRORAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, CreateFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(CreateFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, CreateLinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(CreateLinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, CreateSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(CreateSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, FallocateAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(FallocateAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, FlushFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(FlushFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, ForgetInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ForgetInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, GetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(GetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, GetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(GetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, ListXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ListXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, LookUpInodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(LookUpInodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, MkDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(MkDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, MkNodeAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(MkNodeAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, OpenDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(OpenDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, OpenFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(OpenFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, ReadDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ReadDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadDirPlus",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, ReadDirPlusAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ReadDirPlusAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, ReadFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ReadFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, ReadSymlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ReadSymlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, ReleaseDirHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ReleaseDirHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, ReleaseFileHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(ReleaseFileHandleAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, RemoveXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(RemoveXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Rename",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, RenameAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(RenameAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, RmDirAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(RmDirAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, SetInodeAttributesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(SetInodeAttributesAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, SetXattrAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(SetXattrAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, StatFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(StatFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, SyncFSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(SyncFSAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, SyncFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(SyncFileAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, UnlinkAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(UnlinkAttr))): 5,
			},
		},
		{
			name: "fs_error_category_TOO_MANY_OPEN_FILES_fs_op_WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, TOOMANYOPENFILESAttr, WriteFileAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_error_category", string(TOOMANYOPENFILESAttr)), attribute.String("fs_op", string(WriteFileAttr))): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(5, DEVICEERRORAttr, BatchForgetAttr)
				m.FsOpsErrorCount(2, DEVICEERRORAttr, CreateFileAttr)
				m.FsOpsErrorCount(3, DEVICEERRORAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 8,
				attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(CreateFileAttr))): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.FsOpsErrorCount(-5, DEVICEERRORAttr, BatchForgetAttr)
				m.FsOpsErrorCount(2, DEVICEERRORAttr, BatchForgetAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("fs_error_category", string(DEVICEERRORAttr)), attribute.String("fs_op", string(BatchForgetAttr))): 2},
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
		fsOp      MetricAttr
	}{
		{
			name:      "fs_op_BatchForget",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      BatchForgetAttr,
		},
		{
			name:      "fs_op_CreateFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      CreateFileAttr,
		},
		{
			name:      "fs_op_CreateLink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      CreateLinkAttr,
		},
		{
			name:      "fs_op_CreateSymlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      CreateSymlinkAttr,
		},
		{
			name:      "fs_op_Fallocate",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FallocateAttr,
		},
		{
			name:      "fs_op_FlushFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      FlushFileAttr,
		},
		{
			name:      "fs_op_ForgetInode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      ForgetInodeAttr,
		},
		{
			name:      "fs_op_GetInodeAttributes",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      GetInodeAttributesAttr,
		},
		{
			name:      "fs_op_GetXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      GetXattrAttr,
		},
		{
			name:      "fs_op_ListXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      ListXattrAttr,
		},
		{
			name:      "fs_op_LookUpInode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      LookUpInodeAttr,
		},
		{
			name:      "fs_op_MkDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      MkDirAttr,
		},
		{
			name:      "fs_op_MkNode",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      MkNodeAttr,
		},
		{
			name:      "fs_op_OpenDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      OpenDirAttr,
		},
		{
			name:      "fs_op_OpenFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      OpenFileAttr,
		},
		{
			name:      "fs_op_ReadDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      ReadDirAttr,
		},
		{
			name:      "fs_op_ReadDirPlus",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      ReadDirPlusAttr,
		},
		{
			name:      "fs_op_ReadFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      ReadFileAttr,
		},
		{
			name:      "fs_op_ReadSymlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      ReadSymlinkAttr,
		},
		{
			name:      "fs_op_ReleaseDirHandle",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      ReleaseDirHandleAttr,
		},
		{
			name:      "fs_op_ReleaseFileHandle",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      ReleaseFileHandleAttr,
		},
		{
			name:      "fs_op_RemoveXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      RemoveXattrAttr,
		},
		{
			name:      "fs_op_Rename",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      RenameAttr,
		},
		{
			name:      "fs_op_RmDir",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      RmDirAttr,
		},
		{
			name:      "fs_op_SetInodeAttributes",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      SetInodeAttributesAttr,
		},
		{
			name:      "fs_op_SetXattr",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      SetXattrAttr,
		},
		{
			name:      "fs_op_StatFS",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      StatFSAttr,
		},
		{
			name:      "fs_op_SyncFS",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      SyncFSAttr,
		},
		{
			name:      "fs_op_SyncFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      SyncFileAttr,
		},
		{
			name:      "fs_op_Unlink",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      UnlinkAttr,
		},
		{
			name:      "fs_op_WriteFile",
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
			fsOp:      WriteFileAttr,
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
				m.GcsDownloadBytesCount(5, ParallelAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", string(ParallelAttr))): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, RandomAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", string(RandomAttr))): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, SequentialAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", string(SequentialAttr))): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(5, ParallelAttr)
				m.GcsDownloadBytesCount(2, RandomAttr)
				m.GcsDownloadBytesCount(3, ParallelAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", string(ParallelAttr))): 8,
				attribute.NewSet(attribute.String("read_type", string(RandomAttr))): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(-5, ParallelAttr)
				m.GcsDownloadBytesCount(2, ParallelAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", string(ParallelAttr))): 2},
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
				m.GcsReadCount(5, ParallelAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", string(ParallelAttr))): 5,
			},
		},
		{
			name: "read_type_Random",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, RandomAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", string(RandomAttr))): 5,
			},
		},
		{
			name: "read_type_Sequential",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, SequentialAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", string(SequentialAttr))): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, ParallelAttr)
				m.GcsReadCount(2, RandomAttr)
				m.GcsReadCount(3, ParallelAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", string(ParallelAttr))): 8,
				attribute.NewSet(attribute.String("read_type", string(RandomAttr))): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsReadCount(-5, ParallelAttr)
				m.GcsReadCount(2, ParallelAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("read_type", string(ParallelAttr))): 2},
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
				m.GcsReaderCount(5, ReadHandleAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", string(ReadHandleAttr))): 5,
			},
		},
		{
			name: "io_method_closed",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, ClosedAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", string(ClosedAttr))): 5,
			},
		},
		{
			name: "io_method_opened",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, OpenedAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", string(OpenedAttr))): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, ReadHandleAttr)
				m.GcsReaderCount(2, ClosedAttr)
				m.GcsReaderCount(3, ReadHandleAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("io_method", string(ReadHandleAttr))): 8,
				attribute.NewSet(attribute.String("io_method", string(ClosedAttr))): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(-5, ReadHandleAttr)
				m.GcsReaderCount(2, ReadHandleAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("io_method", string(ReadHandleAttr))): 2},
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
				m.GcsRequestCount(5, ComposeObjectsAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(ComposeObjectsAttr))): 5,
			},
		},
		{
			name: "gcs_method_CopyObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, CopyObjectAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(CopyObjectAttr))): 5,
			},
		},
		{
			name: "gcs_method_CreateAppendableObjectWriter",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, CreateAppendableObjectWriterAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(CreateAppendableObjectWriterAttr))): 5,
			},
		},
		{
			name: "gcs_method_CreateFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, CreateFolderAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(CreateFolderAttr))): 5,
			},
		},
		{
			name: "gcs_method_CreateObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, CreateObjectAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(CreateObjectAttr))): 5,
			},
		},
		{
			name: "gcs_method_CreateObjectChunkWriter",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, CreateObjectChunkWriterAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(CreateObjectChunkWriterAttr))): 5,
			},
		},
		{
			name: "gcs_method_DeleteFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, DeleteFolderAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(DeleteFolderAttr))): 5,
			},
		},
		{
			name: "gcs_method_DeleteObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, DeleteObjectAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(DeleteObjectAttr))): 5,
			},
		},
		{
			name: "gcs_method_FinalizeUpload",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, FinalizeUploadAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(FinalizeUploadAttr))): 5,
			},
		},
		{
			name: "gcs_method_FlushPendingWrites",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, FlushPendingWritesAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(FlushPendingWritesAttr))): 5,
			},
		},
		{
			name: "gcs_method_GetFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, GetFolderAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(GetFolderAttr))): 5,
			},
		},
		{
			name: "gcs_method_ListObjects",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, ListObjectsAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(ListObjectsAttr))): 5,
			},
		},
		{
			name: "gcs_method_MoveObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, MoveObjectAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(MoveObjectAttr))): 5,
			},
		},
		{
			name: "gcs_method_MultiRangeDownloader::Add",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, MultiRangeDownloaderAddAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(MultiRangeDownloaderAddAttr))): 5,
			},
		},
		{
			name: "gcs_method_NewMultiRangeDownloader",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, NewMultiRangeDownloaderAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(NewMultiRangeDownloaderAttr))): 5,
			},
		},
		{
			name: "gcs_method_NewReader",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, NewReaderAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(NewReaderAttr))): 5,
			},
		},
		{
			name: "gcs_method_RenameFolder",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, RenameFolderAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(RenameFolderAttr))): 5,
			},
		},
		{
			name: "gcs_method_StatObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, StatObjectAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(StatObjectAttr))): 5,
			},
		},
		{
			name: "gcs_method_UpdateObject",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, UpdateObjectAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("gcs_method", string(UpdateObjectAttr))): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(5, ComposeObjectsAttr)
				m.GcsRequestCount(2, CopyObjectAttr)
				m.GcsRequestCount(3, ComposeObjectsAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("gcs_method", string(ComposeObjectsAttr))): 8,
				attribute.NewSet(attribute.String("gcs_method", string(CopyObjectAttr))): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsRequestCount(-5, ComposeObjectsAttr)
				m.GcsRequestCount(2, ComposeObjectsAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("gcs_method", string(ComposeObjectsAttr))): 2},
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
		gcsMethod MetricAttr
	}{
		{
			name:      "gcs_method_ComposeObjects",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: ComposeObjectsAttr,
		},
		{
			name:      "gcs_method_CopyObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: CopyObjectAttr,
		},
		{
			name:      "gcs_method_CreateAppendableObjectWriter",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: CreateAppendableObjectWriterAttr,
		},
		{
			name:      "gcs_method_CreateFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: CreateFolderAttr,
		},
		{
			name:      "gcs_method_CreateObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: CreateObjectAttr,
		},
		{
			name:      "gcs_method_CreateObjectChunkWriter",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: CreateObjectChunkWriterAttr,
		},
		{
			name:      "gcs_method_DeleteFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: DeleteFolderAttr,
		},
		{
			name:      "gcs_method_DeleteObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: DeleteObjectAttr,
		},
		{
			name:      "gcs_method_FinalizeUpload",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: FinalizeUploadAttr,
		},
		{
			name:      "gcs_method_FlushPendingWrites",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: FlushPendingWritesAttr,
		},
		{
			name:      "gcs_method_GetFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: GetFolderAttr,
		},
		{
			name:      "gcs_method_ListObjects",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: ListObjectsAttr,
		},
		{
			name:      "gcs_method_MoveObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: MoveObjectAttr,
		},
		{
			name:      "gcs_method_MultiRangeDownloader::Add",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: MultiRangeDownloaderAddAttr,
		},
		{
			name:      "gcs_method_NewMultiRangeDownloader",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: NewMultiRangeDownloaderAttr,
		},
		{
			name:      "gcs_method_NewReader",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: NewReaderAttr,
		},
		{
			name:      "gcs_method_RenameFolder",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: RenameFolderAttr,
		},
		{
			name:      "gcs_method_StatObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: StatObjectAttr,
		},
		{
			name:      "gcs_method_UpdateObject",
			latencies: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
			gcsMethod: UpdateObjectAttr,
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
				m.GcsRetryCount(5, OTHERERRORSAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", string(OTHERERRORSAttr))): 5,
			},
		},
		{
			name: "retry_error_category_STALLED_READ_REQUEST",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(5, STALLEDREADREQUESTAttr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", string(STALLEDREADREQUESTAttr))): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(5, OTHERERRORSAttr)
				m.GcsRetryCount(2, STALLEDREADREQUESTAttr)
				m.GcsRetryCount(3, OTHERERRORSAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("retry_error_category", string(OTHERERRORSAttr))): 8,
				attribute.NewSet(attribute.String("retry_error_category", string(STALLEDREADREQUESTAttr))): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(-5, OTHERERRORSAttr)
				m.GcsRetryCount(2, OTHERERRORSAttr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("retry_error_category", string(OTHERERRORSAttr))): 2},
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
				m.TestUpdownCounterWithAttrs(5, Attr1Attr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("request_type", string(Attr1Attr))): 5,
			},
		},
		{
			name: "request_type_attr2",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(5, Attr2Attr)
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("request_type", string(Attr2Attr))): 5,
			},
		}, {
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(5, Attr1Attr)
				m.TestUpdownCounterWithAttrs(2, Attr2Attr)
				m.TestUpdownCounterWithAttrs(3, Attr1Attr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("request_type", string(Attr1Attr))): 8,
				attribute.NewSet(attribute.String("request_type", string(Attr2Attr))): 2,
			},
		},
		{
			name: "negative_increment",
			f: func(m *otelMetrics) {
				m.TestUpdownCounterWithAttrs(-5, Attr1Attr)
				m.TestUpdownCounterWithAttrs(2, Attr1Attr)
			},
			expected: map[attribute.Set]int64{attribute.NewSet(attribute.String("request_type", string(Attr1Attr))): -3},
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
