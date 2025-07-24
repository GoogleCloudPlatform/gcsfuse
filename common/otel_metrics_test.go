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

func waitForMetricsProcessing() {
	// Currently the metrics processing is synchronous, so there is no waiting.
	// This will change as we will move to asynchronous metrics.
}

func setupOTel(t *testing.T) (*otelMetrics, *metric.ManualReader) {
	t.Helper()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)

	m, err := NewOTelMetrics()
	require.NoError(t, err)
	return m, reader
}

// gatherHistogramMetrics collects all histogram metrics from the reader.
// It returns a map where the key is the metric name, and the value is another map.
// The inner map's key is the set of attributes,
// and the value is the HistogramDataPoint.
func gatherHistogramMetrics(ctx context.Context, t *testing.T, rd *metric.ManualReader) map[string]map[attribute.Set]metricdata.HistogramDataPoint[int64] {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := rd.Collect(ctx, &rm)
	require.NoError(t, err)

	results := make(map[string]map[attribute.Set]metricdata.HistogramDataPoint[int64])

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			// We are interested in Histogram[int64].
			hist, ok := m.Data.(metricdata.Histogram[int64])
			if !ok {
				continue
			}

			metricMap := make(map[attribute.Set]metricdata.HistogramDataPoint[int64])
			for _, dp := range hist.DataPoints {
				if dp.Count == 0 {
					continue
				}

				metricMap[dp.Attributes] = dp
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
// The inner map's key is the set of attributes,
// and the value is the counter's value.
func gatherNonZeroCounterMetrics(ctx context.Context, t *testing.T, rd *metric.ManualReader) map[string]map[attribute.Set]int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := rd.Collect(ctx, &rm)
	require.NoError(t, err)

	results := make(map[string]map[attribute.Set]int64)

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			// We are interested in Sum[int64] which corresponds to int_counter.
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}

			metricMap := make(map[attribute.Set]int64)
			for _, dp := range sum.DataPoints {
				if dp.Value == 0 {
					continue
				}

				// The attribute.Set can be used as a map key directly.
				metricMap[dp.Attributes] = dp.Value
			}

			if len(metricMap) > 0 {
				results[m.Name] = metricMap
			}
		}
	}

	return results
}

func TestFsOpsCount(t *testing.T) {
	fsOps := []string{
		"StatFS", "LookUpInode", "GetInodeAttributes", "SetInodeAttributes", "ForgetInode",
		"BatchForget", "MkDir", "MkNode", "CreateFile", "CreateLink", "CreateSymlink",
		"Rename", "RmDir", "Unlink", "OpenDir", "ReadDir", "ReleaseDirHandle",
		"OpenFile", "ReadFile", "WriteFile", "SyncFile", "FlushFile", "ReleaseFileHandle",
		"ReadSymlink", "RemoveXattr", "GetXattr", "ListXattr", "SetXattr", "Fallocate", "SyncFS",
	}

	for _, op := range fsOps {
		t.Run(op, func(t *testing.T) {
			ctx := context.Background()
			m, rd := setupOTel(t)

			m.OpsCount(context.TODO(), 3, op)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			opsCount, ok := metrics["fs/ops_count"]
			expected := map[attribute.Set]int64{
				attribute.NewSet(attribute.String("fs_op", op)): 3,
			}
			assert.True(t, ok, "fs/ops_count metric not found")
			assert.Equal(t, expected, opsCount)
		})
	}

}

func TestMultipleFSOpsSummed(t *testing.T) {
	ctx := context.Background()
	m, rd := setupOTel(t)

	m.OpsCount(context.TODO(), 5, "BatchForget")
	m.OpsCount(context.TODO(), 2, "CreateFile")
	m.OpsCount(context.TODO(), 3, "BatchForget")

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	opsCount, ok := metrics["fs/ops_count"]
	assert.True(t, ok, "fs/ops_count metric not found")
	expected := map[attribute.Set]int64{
		attribute.NewSet(attribute.String("fs_op", "BatchForget")): 8,
		attribute.NewSet(attribute.String("fs_op", "CreateFile")):  2,
	}
	assert.Equal(t, expected, opsCount)

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
				m, rd := setupOTel(t)

				m.OpsErrorCount(context.TODO(), 5, FSOpsErrorCategory{ErrorCategory: category, FSOps: op})
				waitForMetricsProcessing()

				metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
				opsErrorCount, ok := metrics["fs/ops_error_count"]
				require.True(t, ok, "fs/ops_error_count metric not found")
				expectedKey := attribute.NewSet(
					attribute.String("fs_error_category", category),
					attribute.String("fs_op", op))
				expected := map[attribute.Set]int64{
					expectedKey: 5,
				}
				assert.Equal(t, expected, opsErrorCount)
			})
		}
	}

}

func TestFsOpsErrorCountSummed(t *testing.T) {
	ctx := context.Background()
	m, rd := setupOTel(t)

	m.OpsErrorCount(context.TODO(), 5, FSOpsErrorCategory{ErrorCategory: "IO_ERROR", FSOps: "ReadFile"})
	m.OpsErrorCount(context.TODO(), 3, FSOpsErrorCategory{ErrorCategory: "IO_ERROR", FSOps: "ReadFile"})
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	opsErrorCount, ok := metrics["fs/ops_error_count"]
	assert.True(t, ok, "fs/ops_error_count metric not found")
	expectedKey := attribute.NewSet(
		attribute.String("fs_error_category", "IO_ERROR"),
		attribute.String("fs_op", "ReadFile"),
	)
	assert.Equal(t, map[attribute.Set]int64{expectedKey: 8}, opsErrorCount)
}

func TestFsOpsErrorCountDifferentErrors(t *testing.T) {
	ctx := context.Background()
	m, rd := setupOTel(t)

	m.OpsErrorCount(context.TODO(), 5, FSOpsErrorCategory{ErrorCategory: "IO_ERROR", FSOps: "ReadFile"})
	m.OpsErrorCount(context.TODO(), 2, FSOpsErrorCategory{ErrorCategory: "NETWORK_ERROR", FSOps: "WriteFile"})
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	opsErrorCount, ok := metrics["fs/ops_error_count"]
	assert.True(t, ok, "fs/ops_error_count metric not found")
	expected := map[attribute.Set]int64{
		attribute.NewSet(
			attribute.String("fs_error_category", "IO_ERROR"),
			attribute.String("fs_op", "ReadFile"),
		): 5,
		attribute.NewSet(
			attribute.String("fs_error_category", "NETWORK_ERROR"),
			attribute.String("fs_op", "WriteFile"),
		): 2,
	}
	assert.Equal(t, expected, opsErrorCount)
}

func TestFsOpsErrorCountDifferentErrorsSummed(t *testing.T) {
	ctx := context.Background()
	m, rd := setupOTel(t)

	m.OpsErrorCount(context.TODO(), 5, FSOpsErrorCategory{ErrorCategory: "IO_ERROR", FSOps: "ReadFile"})
	m.OpsErrorCount(context.TODO(), 2, FSOpsErrorCategory{ErrorCategory: "NETWORK_ERROR", FSOps: "WriteFile"})
	m.OpsErrorCount(context.TODO(), 3, FSOpsErrorCategory{ErrorCategory: "IO_ERROR", FSOps: "ReadFile"})
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	opsErrorCount, ok := metrics["fs/ops_error_count"]
	assert.True(t, ok, "fs/ops_error_count metric not found")
	expected := map[attribute.Set]int64{
		attribute.NewSet(
			attribute.String("fs_error_category", "IO_ERROR"),
			attribute.String("fs_op", "ReadFile"),
		): 8,
		attribute.NewSet(
			attribute.String("fs_error_category", "NETWORK_ERROR"),
			attribute.String("fs_op", "WriteFile"),
		): 2,
	}
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
			m, rd := setupOTel(t)
			latency := 123 * time.Microsecond

			m.OpsLatency(ctx, latency, op)
			waitForMetricsProcessing()

			metrics := gatherHistogramMetrics(ctx, t, rd)
			opsLatency, ok := metrics["fs/ops_latency"]
			require.True(t, ok, "fs/ops_latency metric not found")
			expectedKey := attribute.NewSet(attribute.String("fs_op", op))
			dp, ok := opsLatency[expectedKey]
			require.True(t, ok, "DataPoint not found for key: %s", expectedKey)
			assert.Equal(t, uint64(1), dp.Count)
			assert.Equal(t, latency.Microseconds(), dp.Sum)
		})
	}
}

func TestFsOpsLatencySummed(t *testing.T) {
	ctx := context.Background()
	m, rd := setupOTel(t)
	latency1 := 100 * time.Microsecond
	latency2 := 200 * time.Microsecond

	m.OpsLatency(ctx, latency1, "ReadFile")
	m.OpsLatency(ctx, latency2, "ReadFile")
	waitForMetricsProcessing()

	metrics := gatherHistogramMetrics(ctx, t, rd)
	opsLatency, ok := metrics["fs/ops_latency"]
	require.True(t, ok, "fs/ops_latency metric not found")
	dp, ok := opsLatency[attribute.NewSet(attribute.String("fs_op", "ReadFile"))]
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
				m.GCSReadCount(context.TODO(), 5, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 5,
			},
		},
		{
			name: "Random",
			f: func(m *otelMetrics) {
				m.GCSReadCount(context.TODO(), 3, "Random")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 3,
			},
		},
		{
			name: "Parallel",
			f: func(m *otelMetrics) {
				m.GCSReadCount(context.TODO(), 2, "Parallel")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 2,
			},
		},
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GCSReadCount(context.TODO(), 5, "Sequential")
				m.GCSReadCount(context.TODO(), 2, "Random")
				m.GCSReadCount(context.TODO(), 3, "Sequential")
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
			m, rd := setupOTel(t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			readCount, ok := metrics["gcs/read_count"]
			assert.True(t, ok, "gcs/read_count metric not found")
			assert.Equal(t, tc.expected, readCount)
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
				m.GCSDownloadBytesCount(context.TODO(), 500, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 500,
			},
		},
		{
			name: "Random",
			f: func(m *otelMetrics) {
				m.GCSDownloadBytesCount(context.TODO(), 300, "Random")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 300,
			},
		},
		{
			name: "Parallel",
			f: func(m *otelMetrics) {
				m.GCSDownloadBytesCount(context.TODO(), 200, "Parallel")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 200,
			},
		},
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GCSDownloadBytesCount(context.TODO(), 500, "Sequential")
				m.GCSDownloadBytesCount(context.TODO(), 200, "Random")
				m.GCSDownloadBytesCount(context.TODO(), 300, "Sequential")
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
			m, rd := setupOTel(t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			downloadBytes, ok := metrics["gcs/download_bytes_count"]
			assert.True(t, ok, "gcs/download_bytes_count metric not found")
			assert.Equal(t, tc.expected, downloadBytes)
		})
	}
}

func TestGcsReadBytesCount(t *testing.T) {
	ctx := context.Background()
	m, rd := setupOTel(t)

	m.GCSReadBytesCount(context.TODO(), 1024)
	m.GCSReadBytesCount(context.TODO(), 2048)
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	readBytes, ok := metrics["gcs/read_bytes_count"]
	require.True(t, ok, "gcs/read_bytes_count metric not found")
	assert.Equal(t, map[attribute.Set]int64{attribute.NewSet(): 3072}, readBytes)
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
				m.GCSReaderCount(context.TODO(), 5, "opened")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "opened")): 5,
			},
		},
		{
			name: "closed",
			f: func(m *otelMetrics) {
				m.GCSReaderCount(context.TODO(), 3, "closed")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("io_method", "closed")): 3,
			},
		},
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GCSReaderCount(context.TODO(), 5, "opened")
				m.GCSReaderCount(context.TODO(), 2, "closed")
				m.GCSReaderCount(context.TODO(), 3, "opened")
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
			m, rd := setupOTel(t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			readerCount, ok := metrics["gcs/reader_count"]
			assert.True(t, ok, "gcs/reader_count metric not found")
			assert.Equal(t, tc.expected, readerCount)
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
			m, rd := setupOTel(t)

			m.GCSRequestCount(context.TODO(), 5, method)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			requestCount, ok := metrics["gcs/request_count"]
			require.True(t, ok, "gcs/request_count metric not found")
			expectedKey := attribute.NewSet(attribute.String("gcs_method", method))
			expected := map[attribute.Set]int64{
				expectedKey: 5,
			}
			assert.Equal(t, expected, requestCount)
		})
	}
}

func TestGcsRequestCountSummed(t *testing.T) {
	ctx := context.Background()
	m, rd := setupOTel(t)

	m.GCSRequestCount(context.TODO(), 5, "StatObject")
	m.GCSRequestCount(context.TODO(), 3, "StatObject")
	m.GCSRequestCount(context.TODO(), 2, "ListObjects")
	waitForMetricsProcessing()

	metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
	requestCount, ok := metrics["gcs/request_count"]
	assert.True(t, ok, "gcs/request_count metric not found")
	assert.Equal(t, map[attribute.Set]int64{
		attribute.NewSet(attribute.String("gcs_method", "StatObject")):  8,
		attribute.NewSet(attribute.String("gcs_method", "ListObjects")): 2,
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
			m, rd := setupOTel(t)
			latency := 123 * time.Millisecond

			m.GCSRequestLatency(ctx, latency, method)
			waitForMetricsProcessing()

			metrics := gatherHistogramMetrics(ctx, t, rd)
			requestLatencies, ok := metrics["gcs/request_latencies"]
			require.True(t, ok, "gcs/request_latencies metric not found")
			expectedKey := attribute.NewSet(attribute.String("gcs_method", method))
			dp, ok := requestLatencies[expectedKey]
			require.True(t, ok, "DataPoint not found for key: %s", expectedKey)
			assert.Equal(t, uint64(1), dp.Count)
			assert.Equal(t, latency.Milliseconds(), dp.Sum)
		})
	}
}

func TestGcsRequestLatenciesSummed(t *testing.T) {
	ctx := context.Background()
	m, rd := setupOTel(t)
	latency1 := 100 * time.Millisecond
	latency2 := 200 * time.Millisecond

	m.GCSRequestLatency(ctx, latency1, "StatObject")
	m.GCSRequestLatency(ctx, latency2, "StatObject")
	waitForMetricsProcessing()

	metrics := gatherHistogramMetrics(ctx, t, rd)
	requestLatencies, ok := metrics["gcs/request_latencies"]
	require.True(t, ok, "gcs/request_latencies metric not found")
	dp, ok := requestLatencies[attribute.NewSet(attribute.String("gcs_method", "StatObject"))]
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
				m.GCSRetryCount(context.TODO(), 5, "STALLED_READ_REQUEST")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", "STALLED_READ_REQUEST")): 5,
			},
		},
		{
			name: "OTHER_ERRORS",
			f: func(m *otelMetrics) {
				m.GCSRetryCount(context.TODO(), 3, "OTHER_ERRORS")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("retry_error_category", "OTHER_ERRORS")): 3,
			},
		},
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.GCSRetryCount(context.TODO(), 5, "STALLED_READ_REQUEST")
				m.GCSRetryCount(context.TODO(), 2, "OTHER_ERRORS")
				m.GCSRetryCount(context.TODO(), 3, "STALLED_READ_REQUEST")
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
			m, rd := setupOTel(t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			retryCount, ok := metrics["gcs/retry_count"]
			assert.True(t, ok, "gcs/retry_count metric not found")
			assert.Equal(t, tc.expected, retryCount)
		})
	}
}

func TestFileCacheReadCountNew(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "cache_hit_true_sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadCount(context.TODO(), 5, CacheHitReadType{CacheHit: true, ReadType: "Sequential"})
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
				m.FileCacheReadCount(context.TODO(), 5, CacheHitReadType{CacheHit: true, ReadType: "Random"})
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
				m.FileCacheReadCount(context.TODO(), 5, CacheHitReadType{CacheHit: true, ReadType: "Parallel"})
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
				m.FileCacheReadCount(context.TODO(), 5, CacheHitReadType{CacheHit: false, ReadType: "Sequential"})
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
				m.FileCacheReadCount(context.TODO(), 5, CacheHitReadType{CacheHit: false, ReadType: "Random"})
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
				m.FileCacheReadCount(context.TODO(), 5, CacheHitReadType{CacheHit: false, ReadType: "Parallel"})
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
				m.FileCacheReadCount(context.TODO(), 5, CacheHitReadType{CacheHit: true, ReadType: "Sequential"})
				m.FileCacheReadCount(context.TODO(), 2, CacheHitReadType{CacheHit: false, ReadType: "Random"})
				m.FileCacheReadCount(context.TODO(), 3, CacheHitReadType{CacheHit: true, ReadType: "Sequential"})
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
			m, rd := setupOTel(t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			readCount, ok := metrics["file_cache/read_count"]
			assert.True(t, ok, "file_cache/read_count metric not found")
			assert.Equal(t, tc.expected, readCount)
		})
	}
}

func TestFileCacheReadBytesCountNew(t *testing.T) {
	tests := []struct {
		name     string
		f        func(m *otelMetrics)
		expected map[attribute.Set]int64
	}{
		{
			name: "Sequential",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(context.TODO(), 500, "Sequential")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Sequential")): 500,
			},
		},
		{
			name: "Random",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(context.TODO(), 300, "Random")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Random")): 300,
			},
		},
		{
			name: "Parallel",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(context.TODO(), 200, "Parallel")
			},
			expected: map[attribute.Set]int64{
				attribute.NewSet(attribute.String("read_type", "Parallel")): 200,
			},
		},
		{
			name: "multiple_attributes_summed",
			f: func(m *otelMetrics) {
				m.FileCacheReadBytesCount(context.TODO(), 500, "Sequential")
				m.FileCacheReadBytesCount(context.TODO(), 200, "Random")
				m.FileCacheReadBytesCount(context.TODO(), 300, "Sequential")
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
			m, rd := setupOTel(t)

			tc.f(m)
			waitForMetricsProcessing()

			metrics := gatherNonZeroCounterMetrics(ctx, t, rd)
			readBytesCount, ok := metrics["file_cache/read_bytes_count"]
			assert.True(t, ok, "file_cache/read_bytes_count metric not found")
			assert.Equal(t, tc.expected, readBytesCount)
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
			m, rd := setupOTel(t)
			var totalLatency time.Duration

			for _, latency := range tc.latencies {
				m.FileCacheReadLatency(ctx, latency, tc.cacheHit)
				totalLatency += latency
			}
			waitForMetricsProcessing()

			metrics := gatherHistogramMetrics(ctx, t, rd)
			readLatencies, ok := metrics["file_cache/read_latencies"]
			require.True(t, ok, "file_cache/read_latencies metric not found")
			expectedKey := attribute.NewSet(attribute.Bool("cache_hit", tc.cacheHit))
			dp, ok := readLatencies[expectedKey]
			require.True(t, ok, "DataPoint not found for key: %s", expectedKey)
			assert.Equal(t, uint64(len(tc.latencies)), dp.Count)
			assert.Equal(t, totalLatency.Microseconds(), dp.Sum)
		})
	}
}
