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

package common

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

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
// The inner map's key is a sorted, semicolon-separated string of attributes,
// and the value is the HistogramDataPoint.
func gatherHistogramMetrics(ctx context.Context, t *testing.T, rd *metric.ManualReader) map[string]map[string]metricdata.HistogramDataPoint[int64] {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := rd.Collect(ctx, &rm)
	require.NoError(t, err)

	results := make(map[string]map[string]metricdata.HistogramDataPoint[int64])

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			// We are interested in Histogram[int64].
			hist, ok := m.Data.(metricdata.Histogram[int64])
			if !ok {
				continue
			}

			metricMap := make(map[string]metricdata.HistogramDataPoint[int64])
			for _, dp := range hist.DataPoints {
				if dp.Count == 0 {
					continue
				}

				var parts []string
				for _, kv := range dp.Attributes.ToSlice() {
					parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
				}
				sort.Strings(parts)
				key := strings.Join(parts, ";")

				metricMap[key] = dp
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
// The inner map's key is a sorted, semicolon-separated string of attributes,
// and the value is the counter's value.
func gatherNonZeroCounterMetrics(ctx context.Context, t *testing.T, rd *metric.ManualReader) map[string]map[string]int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := rd.Collect(ctx, &rm)
	require.NoError(t, err)

	results := make(map[string]map[string]int64)

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			// We are interested in Sum[int64] which corresponds to int_counter.
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}

			metricMap := make(map[string]int64)
			for _, dp := range sum.DataPoints {
				if dp.Value == 0 {
					continue
				}

				var parts []string
				for _, kv := range dp.Attributes.ToSlice() {
					parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
				}
				sort.Strings(parts)
				key := strings.Join(parts, ";")

				metricMap[key] = dp.Value
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
		expected map[string]int64
	}{
		{
			name: "StatFS",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "StatFS")
			},
			expected: map[string]int64{
				"fs_op=StatFS": 3,
			},
		},
		{
			name: "LookUpInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "LookUpInode")
			},
			expected: map[string]int64{
				"fs_op=LookUpInode": 3,
			},
		},
		{
			name: "GetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "GetInodeAttributes")
			},
			expected: map[string]int64{
				"fs_op=GetInodeAttributes": 3,
			},
		},
		{
			name: "SetInodeAttributes",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "SetInodeAttributes")
			},
			expected: map[string]int64{
				"fs_op=SetInodeAttributes": 3,
			},
		},
		{
			name: "ForgetInode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ForgetInode")
			},
			expected: map[string]int64{
				"fs_op=ForgetInode": 3,
			},
		},
		{
			name: "BatchForget",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "BatchForget")
			},
			expected: map[string]int64{
				"fs_op=BatchForget": 3,
			},
		},
		{
			name: "MkDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "MkDir")
			},
			expected: map[string]int64{
				"fs_op=MkDir": 3,
			},
		},
		{
			name: "MkNode",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "MkNode")
			},
			expected: map[string]int64{
				"fs_op=MkNode": 3,
			},
		},
		{
			name: "CreateFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "CreateFile")
			},
			expected: map[string]int64{
				"fs_op=CreateFile": 3,
			},
		},
		{
			name: "CreateLink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "CreateLink")
			},
			expected: map[string]int64{
				"fs_op=CreateLink": 3,
			},
		},
		{
			name: "CreateSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "CreateSymlink")
			},
			expected: map[string]int64{
				"fs_op=CreateSymlink": 3,
			},
		},
		{
			name: "Rename",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "Rename")
			},
			expected: map[string]int64{
				"fs_op=Rename": 3,
			},
		},
		{
			name: "RmDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "RmDir")
			},
			expected: map[string]int64{
				"fs_op=RmDir": 3,
			},
		},
		{
			name: "Unlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "Unlink")
			},
			expected: map[string]int64{
				"fs_op=Unlink": 3,
			},
		},
		{
			name: "OpenDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "OpenDir")
			},
			expected: map[string]int64{
				"fs_op=OpenDir": 3,
			},
		},
		{
			name: "ReadDir",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ReadDir")
			},
			expected: map[string]int64{
				"fs_op=ReadDir": 3,
			},
		},
		{
			name: "ReleaseDirHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ReleaseDirHandle")
			},
			expected: map[string]int64{
				"fs_op=ReleaseDirHandle": 3,
			},
		},
		{
			name: "OpenFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "OpenFile")
			},
			expected: map[string]int64{
				"fs_op=OpenFile": 3,
			},
		},
		{
			name: "ReadFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ReadFile")
			},
			expected: map[string]int64{
				"fs_op=ReadFile": 3,
			},
		},
		{
			name: "WriteFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "WriteFile")
			},
			expected: map[string]int64{
				"fs_op=WriteFile": 3,
			},
		},
		{
			name: "SyncFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "SyncFile")
			},
			expected: map[string]int64{
				"fs_op=SyncFile": 3,
			},
		},
		{
			name: "FlushFile",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "FlushFile")
			},
			expected: map[string]int64{
				"fs_op=FlushFile": 3,
			},
		},
		{
			name: "ReleaseFileHandle",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ReleaseFileHandle")
			},
			expected: map[string]int64{
				"fs_op=ReleaseFileHandle": 3,
			},
		},
		{
			name: "ReadSymlink",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ReadSymlink")
			},
			expected: map[string]int64{
				"fs_op=ReadSymlink": 3,
			},
		},
		{
			name: "RemoveXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "RemoveXattr")
			},
			expected: map[string]int64{
				"fs_op=RemoveXattr": 3,
			},
		},
		{
			name: "GetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "GetXattr")
			},
			expected: map[string]int64{
				"fs_op=GetXattr": 3,
			},
		},
		{
			name: "ListXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "ListXattr")
			},
			expected: map[string]int64{
				"fs_op=ListXattr": 3,
			},
		},
		{
			name: "SetXattr",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "SetXattr")
			},
			expected: map[string]int64{
				"fs_op=SetXattr": 3,
			},
		},
		{
			name: "Fallocate",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "Fallocate")
			},
			expected: map[string]int64{
				"fs_op=Fallocate": 3,
			},
		},
		{
			name: "SyncFS",
			f: func(m *otelMetrics) {
				m.FsOpsCount(3, "SyncFS")
			},
			expected: map[string]int64{
				"fs_op=SyncFS": 3,
			},
		},
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FsOpsCount(5, "BatchForget")
				m.FsOpsCount(2, "CreateFile")
				m.FsOpsCount(3, "BatchForget")
			},
			expected: map[string]int64{
				"fs_op=BatchForget": 8,
				"fs_op=CreateFile":  2,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			m, rd := setupOTel(ctx, t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			opsCount, ok := metrics["fs/ops_count"]
			assert.True(t, ok, "fs/ops_count metric not found")
			assert.Equal(t, tc.expected, opsCount)
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
				m, rd := setupOTel(ctx, t)

				m.FsOpsErrorCount(5, category, op)
				waitForMetricsProcessing()

				metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
				opsErrorCount, ok := metrics["fs/ops_error_count"]
				require.True(t, ok, "fs/ops_error_count metric not found")
				expectedKey := fmt.Sprintf("fs_error_category=%s;fs_op=%s", category, op)
				expected := map[string]int64{
					expectedKey: 5,
				}
				assert.Equal(t, expected, opsErrorCount)
			})
		}
	}

}

func TestFsOpsErrorCountSummed(t *testing.T) {
	ctx := context.Background()
	m, rd := setupOTel(ctx, t)

	m.FsOpsErrorCount(5, "IO_ERROR", "ReadFile")
	m.FsOpsErrorCount(3, "IO_ERROR", "ReadFile")
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	opsErrorCount, ok := metrics["fs/ops_error_count"]
	assert.True(t, ok, "fs/ops_error_count metric not found")
	assert.Equal(t, map[string]int64{"fs_error_category=IO_ERROR;fs_op=ReadFile": 8}, opsErrorCount)
}
func TestFsOpsErrorCountDifferentErrors(t *testing.T) {
	ctx := context.Background()
	m, rd := setupOTel(ctx, t)

	m.FsOpsErrorCount(5, "IO_ERROR", "ReadFile")
	m.FsOpsErrorCount(2, "NETWORK_ERROR", "WriteFile")
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	opsErrorCount, ok := metrics["fs/ops_error_count"]
	assert.True(t, ok, "fs/ops_error_count metric not found")
	assert.Equal(t, map[string]int64{"fs_error_category=IO_ERROR;fs_op=ReadFile": 5, "fs_error_category=NETWORK_ERROR;fs_op=WriteFile": 2}, opsErrorCount)
}
func TestFsOpsErrorCountDifferentErrorsSummed(t *testing.T) {
	ctx := context.Background()
	m, rd := setupOTel(ctx, t)

	m.FsOpsErrorCount(5, "IO_ERROR", "ReadFile")
	m.FsOpsErrorCount(2, "NETWORK_ERROR", "WriteFile")
	m.FsOpsErrorCount(3, "IO_ERROR", "ReadFile")
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	opsErrorCount, ok := metrics["fs/ops_error_count"]
	assert.True(t, ok, "fs/ops_error_count metric not found")
	assert.Equal(t, map[string]int64{"fs_error_category=IO_ERROR;fs_op=ReadFile": 8, "fs_error_category=NETWORK_ERROR;fs_op=WriteFile": 2}, opsErrorCount)
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
			m, rd := setupOTel(ctx, t)
			latency := 123 * time.Microsecond

			m.FsOpsLatency(ctx, latency, op)
			waitForMetricsProcessing()

			metrics := gatherHistogramMetrics(ctx, t, rd)
			opsLatency, ok := metrics["fs/ops_latency"]
			require.True(t, ok, "fs/ops_latency metric not found")
			expectedKey := fmt.Sprintf("fs_op=%s", op)
			dp, ok := opsLatency[expectedKey]
			require.True(t, ok, "DataPoint not found for key: %s", expectedKey)
			assert.Equal(t, uint64(1), dp.Count)
			assert.Equal(t, latency.Microseconds(), dp.Sum)
		})
	}
}

func TestFsOpsLatencySummed(t *testing.T) {
	ctx := context.Background()
	m, rd := setupOTel(ctx, t)
	latency1 := 100 * time.Microsecond
	latency2 := 200 * time.Microsecond

	m.FsOpsLatency(ctx, latency1, "ReadFile")
	m.FsOpsLatency(ctx, latency2, "ReadFile")
	waitForMetricsProcessing()

	metrics := gatherHistogramMetrics(ctx, t, rd)
	opsLatency, ok := metrics["fs/ops_latency"]
	require.True(t, ok, "fs/ops_latency metric not found")
	dp, ok := opsLatency["fs_op=ReadFile"]
	require.True(t, ok, "DataPoint not found for key: fs_op=ReadFile")
	assert.Equal(t, uint64(2), dp.Count)
	assert.Equal(t, latency1.Microseconds()+latency2.Microseconds(), dp.Sum)
}

func waitForMetricsProcessing() {
	time.Sleep(time.Millisecond)
}
