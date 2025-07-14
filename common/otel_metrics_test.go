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

package common

import (
	"context"
	"fmt"
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

			metricMap := make(metricHistogramMap) // Changed key type to map[string]interface{}
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

			metricMap := make(metricValueMap) // Changed key type to map[string]interface{}
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

func TestFsOpsCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "StatFS")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "StatFS")): 3,
			},
		},
		{
			name: "LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "LookUpInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "LookUpInode")): 3,
			},
		},
		{
			name: "GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "GetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "GetInodeAttributes")): 3,
			},
		},
		{
			name: "SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "SetInodeAttributes")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SetInodeAttributes")): 3,
			},
		},
		{
			name: "ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ForgetInode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ForgetInode")): 3,
			},
		},
		{
			name: "BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "BatchForget")): 3,
			},
		},
		{
			name: "MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "MkDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "MkDir")): 3,
			},
		},
		{
			name: "MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "MkNode")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "MkNode")): 3,
			},
		},
		{
			name: "CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "CreateFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateFile")): 3,
			},
		},
		{
			name: "CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "CreateLink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateLink")): 3,
			},
		},
		{
			name: "CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "CreateSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "CreateSymlink")): 3,
			},
		},
		{
			name: "Rename",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "Rename")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Rename")): 3,
			},
		},
		{
			name: "RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "RmDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "RmDir")): 3,
			},
		},
		{
			name: "Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "Unlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Unlink")): 3,
			},
		},
		{
			name: "OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "OpenDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "OpenDir")): 3,
			},
		},
		{
			name: "ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ReadDir")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadDir")): 3,
			},
		},
		{
			name: "ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ReleaseDirHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReleaseDirHandle")): 3,
			},
		},
		{
			name: "OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "OpenFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "OpenFile")): 3,
			},
		},
		{
			name: "ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ReadFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadFile")): 3,
			},
		},
		{
			name: "WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "WriteFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "WriteFile")): 3,
			},
		},
		{
			name: "SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "SyncFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SyncFile")): 3,
			},
		},
		{
			name: "FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "FlushFile")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "FlushFile")): 3,
			},
		},
		{
			name: "ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ReleaseFileHandle")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReleaseFileHandle")): 3,
			},
		},
		{
			name: "ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ReadSymlink")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ReadSymlink")): 3,
			},
		},
		{
			name: "RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "RemoveXattr")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "RemoveXattr")): 3,
			},
		},
		{
			name: "GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "GetXattr")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "GetXattr")): 3,
			},
		},
		{
			name: "ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ListXattr")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "ListXattr")): 3,
			},
		},
		{
			name: "SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "SetXattr")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SetXattr")): 3,
			},
		},
		{
			name: "Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "Fallocate")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "Fallocate")): 3,
			},
		},
		{
			name: "SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "SyncFS")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "SyncFS")): 3,
			},
		},
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "BatchForget")
				m.FsOpsCount(2, "CreateFile")
				m.FsOpsCount(3, "BatchForget")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", "BatchForget")): 8,
				attribute.NewSet(attribute.String("fs_op", "CreateFile")):  2,
			},
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
			opsCount, ok := metrics["fs/ops_count"]
			assert.True(t, ok, "fs/ops_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, opsCount, expectedMap)
		})
	}

}

func TestFsOpsErrorCount(t *testing.T) {
	fsOps := []string{
		"StatFS", "LookUpInode", "GetInodeAttributes", "SetInodeAttributes", "ForgetInode",
		"BatchForget", "MkDir", "MkNode", "CreateFile", "CreateLink", "CreateSymlink",
		"Rename", "RmDir", "Unlink", "OpenDir", "ReadDir", "ReleaseDirHandle",
		"OpenFile", "ReadFile", "WriteFile", "SyncFile", "FlushFile", "ReleaseFileHandle",
		"ReadSymlink", "RemoveXattr", "GetXattr", "ListXattr", "SetXattr", "Fallocate", "SyncFS",
	}
	fsErrorCategories := []string{
		"DEVICE_ERROR", "DIR_NOT_EMPTY", "FILE_EXISTS", "FILE_DIR_ERROR", "NOT_IMPLEMENTED",
		"IO_ERROR", "INTERRUPT_ERROR", "INVALID_ARGUMENT", "INVALID_OPERATION", "MISC_ERROR",
		"NETWORK_ERROR", "NO_FILE_OR_DIR", "NOT_A_DIR", "PERM_ERROR",
		"PROCESS_RESOURCE_MGMT_ERROR", "TOO_MANY_OPEN_FILES",
	}

	for _, op := range fsOps {
		for _, category := range fsErrorCategories {
			op, category := op, category
			t.Run(fmt.Sprintf("%s_%s", op, category), func(t *testing.T) {
				ctx := context.Background()
				encoder := attribute.DefaultEncoder()
				m, rd := setupOTel(ctx, t)

				m.FsOpsErrorCount(5, category, op)
				waitForMetricsProcessing()

				metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
				opsErrorCount, ok := metrics["fs/ops_error_count"]
				require.True(t, ok, "fs/ops_error_count metric not found")
				s := attribute.NewSet(
					attribute.String("fs_error_category", category), attribute.String("fs_op", op))
				expected := map[string]int64{
					s.Encoded(encoder): 5,
				}
				assert.Equal(t, expected, opsErrorCount)
			})
		}
	}

}

func TestFsOpsErrorCountSummed(t *testing.T) {
	ctx := context.Background()
	encoder := attribute.DefaultEncoder()
	m, rd := setupOTel(ctx, t)

	m.FsOpsErrorCount(5, "IO_ERROR", "ReadFile")
	m.FsOpsErrorCount(3, "IO_ERROR", "ReadFile")
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	opsErrorCount, ok := metrics["fs/ops_error_count"]
	assert.True(t, ok, "fs/ops_error_count metric not found")
	s := attribute.NewSet(
		attribute.String("fs_error_category", "IO_ERROR"),
		attribute.String("fs_op", "ReadFile"),
	)
	assert.Equal(t, opsErrorCount, map[string]int64{s.Encoded(encoder): 8})
}

func TestFsOpsErrorCountDifferentErrors(t *testing.T) {
	ctx := context.Background()
	encoder := attribute.DefaultEncoder()
	m, rd := setupOTel(ctx, t)

	m.FsOpsErrorCount(5, "IO_ERROR", "ReadFile")
	m.FsOpsErrorCount(2, "NETWORK_ERROR", "WriteFile")
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	opsErrorCount, ok := metrics["fs/ops_error_count"]
	assert.True(t, ok, "fs/ops_error_count metric not found")
	s1 := attribute.NewSet(
		attribute.String("fs_error_category", "IO_ERROR"),
		attribute.String("fs_op", "ReadFile"),
	)
	s2 := attribute.NewSet(
		attribute.String("fs_error_category", "NETWORK_ERROR"),
		attribute.String("fs_op", "WriteFile"),
	)
	expected := map[string]int64{s1.Encoded(encoder): 5, s2.Encoded(encoder): 2}
	assert.Equal(t, opsErrorCount, expected)
}

func TestFsOpsErrorCountDifferentErrorsSummed(t *testing.T) {
	ctx := context.Background()
	encoder := attribute.DefaultEncoder()
	m, rd := setupOTel(ctx, t)

	m.FsOpsErrorCount(5, "IO_ERROR", "ReadFile")
	m.FsOpsErrorCount(2, "NETWORK_ERROR", "WriteFile")
	m.FsOpsErrorCount(3, "IO_ERROR", "ReadFile")
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	opsErrorCount, ok := metrics["fs/ops_error_count"]
	assert.True(t, ok, "fs/ops_error_count metric not found")
	s1 := attribute.NewSet(
		attribute.String("fs_error_category", "IO_ERROR"),
		attribute.String("fs_op", "ReadFile"),
	)
	s2 := attribute.NewSet(
		attribute.String("fs_error_category", "NETWORK_ERROR"),
		attribute.String("fs_op", "WriteFile"),
	)
	expected := map[string]int64{s1.Encoded(encoder): 8, s2.Encoded(encoder): 2}
	assert.Equal(t, expected, opsErrorCount)
}

func TestFsOpsLatency(t *testing.T) {
	fsOps := []string{
		"StatFS", "LookUpInode", "GetInodeAttributes", "SetInodeAttributes", "ForgetInode",
		"BatchForget", "MkDir", "MkNode", "CreateFile", "CreateLink", "CreateSymlink",
		"Rename", "RmDir", "Unlink", "OpenDir", "ReadDir", "ReleaseDirHandle",
		"OpenFile", "ReadFile", "WriteFile", "SyncFile", "FlushFile", "ReleaseFileHandle",
		"ReadSymlink", "RemoveXattr", "GetXattr", "ListXattr", "SetXattr", "Fallocate", "SyncFS",
	}

	for _, op := range fsOps {
		op := op
		t.Run(op, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)
			latency := 123 * time.Microsecond

			m.FsOpsLatency(ctx, latency, op)
			waitForMetricsProcessing()

			metrics := gatherHistogramMetrics(ctx, t, rd)
			opsLatency, ok := metrics["fs/ops_latency"]
			require.True(t, ok, "fs/ops_latency metric not found")
			s := attribute.NewSet(attribute.String("fs_op", op))
			expectedKey := s.Encoded(encoder)
			dp, ok := opsLatency[expectedKey]
			require.True(t, ok, "DataPoint not found for key: %s", expectedKey)
			assert.Equal(t, uint64(1), dp.Count)
			assert.Equal(t, latency.Microseconds(), dp.Sum)
		})
	}
}

func TestFsOpsLatencySummed(t *testing.T) {
	ctx := context.Background()
	encoder := attribute.DefaultEncoder()
	m, rd := setupOTel(ctx, t)
	latency1 := 100 * time.Microsecond
	latency2 := 200 * time.Microsecond

	m.FsOpsLatency(ctx, latency1, "ReadFile")
	m.FsOpsLatency(ctx, latency2, "ReadFile")
	waitForMetricsProcessing()

	metrics := gatherHistogramMetrics(ctx, t, rd)
	opsLatency, ok := metrics["fs/ops_latency"]
	require.True(t, ok, "fs/ops_latency metric not found")
	s := attribute.NewSet(attribute.String("fs_op", "ReadFile"))
	dp, ok := opsLatency[s.Encoded(encoder)]
	require.True(t, ok, "DataPoint not found for key: fs_op=ReadFile")
	assert.Equal(t, uint64(2), dp.Count)
	assert.Equal(t, latency1.Microseconds()+latency2.Microseconds(), dp.Sum)
}

func TestGcsReadCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "Sequential",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "Random",
			f: func(m *otelMetrics) {
				m.GcsReadCount(3, "Random")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 3,
			},
		},
		{
			name: "Parallel",
			f: func(m *otelMetrics) {
				m.GcsReadCount(2, "Parallel")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 2,
			},
		},
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsReadCount(5, "Sequential")
				m.GcsReadCount(2, "Random")
				m.GcsReadCount(3, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 8,
				attribute.NewSet(attribute.String("read_type", "Random")):     2,
			},
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
			readCount, ok := metrics["gcs/read_count"]
			assert.True(t, ok, "gcs/read_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, readCount, expectedMap)
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
			name: "Sequential",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(500, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 500,
			},
		},
		{
			name: "Random",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(300, "Random")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 300,
			},
		},
		{
			name: "Parallel",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(200, "Parallel")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 200,
			},
		},
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsDownloadBytesCount(500, "Sequential")
				m.GcsDownloadBytesCount(200, "Random")
				m.GcsDownloadBytesCount(300, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 800,
				attribute.NewSet(attribute.String("read_type", "Random")):     200,
			},
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
			downloadBytes, ok := metrics["gcs/download_bytes_count"]
			assert.True(t, ok, "gcs/download_bytes_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, downloadBytes, expectedMap)
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
	readBytes, ok := metrics["gcs/read_bytes_count"]
	require.True(t, ok, "gcs/read_bytes_count metric not found")
	s := attribute.NewSet()
	assert.Equal(t, map[string]int64{s.Encoded(encoder): 3072}, readBytes)
}

func TestGcsReaderCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "opened",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, "opened")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "opened")): 5,
			},
		},
		{
			name: "closed",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(3, "closed")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "closed")): 3,
			},
		},
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsReaderCount(5, "opened")
				m.GcsReaderCount(2, "closed")
				m.GcsReaderCount(3, "opened")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "opened")): 8,
				attribute.NewSet(attribute.String("io_method", "closed")): 2,
			},
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
			readerCount, ok := metrics["gcs/reader_count"]
			assert.True(t, ok, "gcs/reader_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, readerCount, expectedMap)
		})
	}
}

func TestGcsRequestCount(t *testing.T) {
	gcsMethods := []string{
		"MultiRangeDownloader::Add", "ComposeObjects", "CreateFolder", "CreateObjectChunkWriter",
		"DeleteFolder", "DeleteObject", "FinalizeUpload", "GetFolder", "ListObjects",
		"MoveObject", "NewReader", "RenameFolder", "StatObject", "UpdateObject",
	}

	for _, method := range gcsMethods {
		method := method
		t.Run(method, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)

			m.GcsRequestCount(5, method)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			requestCount, ok := metrics["gcs/request_count"]
			require.True(t, ok, "gcs/request_count metric not found")
			s := attribute.NewSet(attribute.String("gcs_method", method))
			expectedKey := s.Encoded(encoder)
			expected := map[string]int64{
				expectedKey: 5,
			}
			assert.Equal(t, expected, requestCount)
		})
	}
}

func TestGcsRequestCountSummed(t *testing.T) {
	ctx := context.Background()
	encoder := attribute.DefaultEncoder()
	m, rd := setupOTel(ctx, t)

	m.GcsRequestCount(5, "StatObject")
	m.GcsRequestCount(3, "StatObject")
	m.GcsRequestCount(2, "ListObjects")
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	requestCount, ok := metrics["gcs/request_count"]
	assert.True(t, ok, "gcs/request_count metric not found")
	s1 := attribute.NewSet(attribute.String("gcs_method", "StatObject"))
	s2 := attribute.NewSet(attribute.String("gcs_method", "ListObjects"))
	assert.Equal(t, map[string]int64{
		s1.Encoded(encoder): 8,
		s2.Encoded(encoder): 2,
	}, requestCount)
}

func TestGcsRequestLatencies(t *testing.T) {
	gcsMethods := []string{
		"MultiRangeDownloader::Add", "ComposeObjects", "CreateFolder", "CreateObjectChunkWriter",
		"DeleteFolder", "DeleteObject", "FinalizeUpload", "GetFolder", "ListObjects",
		"MoveObject", "NewReader", "RenameFolder", "StatObject", "UpdateObject",
	}

	for _, method := range gcsMethods {
		method := method
		t.Run(method, func(t *testing.T) {
			ctx := context.Background()
			encoder := attribute.DefaultEncoder()
			m, rd := setupOTel(ctx, t)
			latency := 123 * time.Millisecond

			m.GcsRequestLatencies(ctx, latency, method)
			waitForMetricsProcessing()

			metrics := gatherHistogramMetrics(ctx, t, rd)
			requestLatencies, ok := metrics["gcs/request_latencies"]
			require.True(t, ok, "gcs/request_latencies metric not found")
			s := attribute.NewSet(attribute.String("gcs_method", method))
			expectedKey := s.Encoded(encoder)
			dp, ok := requestLatencies[expectedKey]
			require.True(t, ok, "DataPoint not found for key: %s", expectedKey)
			assert.Equal(t, uint64(1), dp.Count)
			assert.Equal(t, latency.Milliseconds(), dp.Sum)
		})
	}
}

func TestGcsRequestLatenciesSummed(t *testing.T) {
	ctx := context.Background()
	encoder := attribute.DefaultEncoder()
	m, rd := setupOTel(ctx, t)
	latency1 := 100 * time.Millisecond
	latency2 := 200 * time.Millisecond

	m.GcsRequestLatencies(ctx, latency1, "StatObject")
	m.GcsRequestLatencies(ctx, latency2, "StatObject")
	waitForMetricsProcessing()

	metrics := gatherHistogramMetrics(ctx, t, rd)
	requestLatencies, ok := metrics["gcs/request_latencies"]
	require.True(t, ok, "gcs/request_latencies metric not found")
	s := attribute.NewSet(attribute.String("gcs_method", "StatObject"))
	dp, ok := requestLatencies[s.Encoded(encoder)]
	require.True(t, ok, "DataPoint not found for key: gcs_method=StatObject")
	assert.Equal(t, uint64(2), dp.Count)
	assert.Equal(t, latency1.Milliseconds()+latency2.Milliseconds(), dp.Sum)
}

func TestGcsRetryCount(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "STALLED_READ_REQUEST",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(5, "STALLED_READ_REQUEST")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", "STALLED_READ_REQUEST")): 5,
			},
		},
		{
			name: "OTHER_ERRORS",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(3, "OTHER_ERRORS")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", "OTHER_ERRORS")): 3,
			},
		},
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GcsRetryCount(5, "STALLED_READ_REQUEST")
				m.GcsRetryCount(2, "OTHER_ERRORS")
				m.GcsRetryCount(3, "STALLED_READ_REQUEST")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", "STALLED_READ_REQUEST")): 8,
				attribute.NewSet(attribute.String("retry_error_category", "OTHER_ERRORS")):         2,
			},
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
			retryCount, ok := metrics["gcs/retry_count"]
			assert.True(t, ok, "gcs/retry_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, retryCount)
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
			name: "cache_hit_true_sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(
					attribute.Bool("cache_hit", true),
					attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "cache_hit_true_random",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, "Random")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(
					attribute.Bool("cache_hit", true),
					attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "cache_hit_true_parallel",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, "Parallel")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(
					attribute.Bool("cache_hit", true),
					attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "cache_hit_false_sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(
					attribute.Bool("cache_hit", false),
					attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "cache_hit_false_random",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, "Random")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(
					attribute.Bool("cache_hit", false),
					attribute.String("read_type", "Random")): 5,
			},
		},
		{
			name: "cache_hit_false_parallel",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, false, "Parallel")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(
					attribute.Bool("cache_hit", false),
					attribute.String("read_type", "Parallel")): 5,
			},
		},
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(5, true, "Sequential")
				m.FileCacheReadCount(2, false, "Random")
				m.FileCacheReadCount(3, true, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.Bool("cache_hit", false), attribute.String("read_type", "Random")):    2,
				attribute.NewSet(attribute.Bool("cache_hit", true), attribute.String("read_type", "Sequential")): 8,
			},
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
			readCount, ok := metrics["file_cache/read_count"]
			assert.True(t, ok, "file_cache/read_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, expectedMap, readCount)
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
			name: "Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(500, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 500,
			},
		},
		{
			name: "Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(300, "Random")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 300,
			},
		},
		{
			name: "Parallel",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(200, "Parallel")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 200,
			},
		},
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(500, "Sequential")
				m.FileCacheReadBytesCount(200, "Random")
				m.FileCacheReadBytesCount(300, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")):     200,
				attribute.NewSet(attribute.String("read_type", "Sequential")): 800,
			},
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
			readBytesCount, ok := metrics["file_cache/read_bytes_count"]
			assert.True(t, ok, "file_cache/read_bytes_count metric not found")
			expectedMap := make(map[string]int64)
			for k, v := range tc.expected {
				expectedMap[k.Encoded(encoder)] = v
			}
			assert.Equal(t, readBytesCount, expectedMap)
		})
	}
}

func TestFileCacheReadLatencies(t *testing.T) {
	tests := []struct {
		name      string
		cacheHit  bool
		latencies []time.Duration
	}{
		{
			name:      "cache_hit_true",
			cacheHit:  true,
			latencies: []time.Duration{100 * time.Microsecond, 200 * time.Microsecond},
		},
		{
			name:      "cache_hit_false",
			cacheHit:  false,
			latencies: []time.Duration{300 * time.Microsecond, 400 * time.Microsecond},
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
			readLatencies, ok := metrics["file_cache/read_latencies"]
			require.True(t, ok, "file_cache/read_latencies metric not found")
			s := attribute.NewSet(attribute.Bool("cache_hit", tc.cacheHit))
			expectedKey := s.Encoded(encoder)
			dp, ok := readLatencies[expectedKey]
			require.True(t, ok, "DataPoint not found for key: %s", expectedKey)
			assert.Equal(t, uint64(len(tc.latencies)), dp.Count)
			assert.Equal(t, totalLatency.Microseconds(), dp.Sum)
		})
	}
}
