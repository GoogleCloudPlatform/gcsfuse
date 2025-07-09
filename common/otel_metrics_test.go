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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// Helper to wait for async metric processing.
// This is a hack. A better solution would be to have a Flush/Close method on otelMetrics.
func waitForMetrics() {
	time.Sleep(100 * time.Millisecond)
}


func TestFsOpsCount(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	
	
	m.FsOpsCount(
		5,"BatchForget")m.FsOpsCount(
		2,"CreateFile")m.FsOpsCount(
		3,"BatchForget")

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "fs/ops_count" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric fs/ops_count not found")
	assert.Equal(t, "The cumulative number of ops processed by the file system.", foundMetric.Description)

	sum, ok := foundMetric.Data.(metricdata.Sum[int64])
	assert.True(t, ok, "metric fs/ops_count is not a Sum")
	dataPoints := sum.DataPoints

	assert.Len(t, dataPoints, 2)
	expected := map[string]struct{ val int64; count uint64 }{
		"fs_op=BatchForget": {val: 8, count: 2},
		"fs_op=CreateFile": {val: 2, count: 1},
	}for _, dp := range dataPoints {
		var parts []string
		dp.Attributes.Range(func(kv attribute.KeyValue) bool {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
			return true
		})
		sort.Strings(parts)
		key := strings.Join(parts, ";")

		e, ok := expected[key]
		assert.True(t, ok, "unexpected attribute set: %s", key)
		assert.Equal(t, e.val, dp.Value, "value mismatch for %s", key)
		delete(expected, key)
	}
	assert.Empty(t, expected, "not all expected attribute sets were found")
}

func TestFsOpsLatency(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	
	
	m.FsOpsLatency(
		ctx, 123*time.Microsecond,"BatchForget")m.FsOpsLatency(
		ctx, 456*time.Microsecond,"CreateFile")m.FsOpsLatency(
		ctx, 789*time.Microsecond,"BatchForget")

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "fs/ops_latency" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric fs/ops_latency not found")
	assert.Equal(t, "The cumulative distribution of file system operation latencies", foundMetric.Description)

	hist, ok := foundMetric.Data.(metricdata.Histogram[int64])
	assert.True(t, ok, "metric fs/ops_latency is not a Histogram")
	dataPoints := hist.DataPoints

	assert.Len(t, dataPoints, 2)
	expected := map[string]struct{ val int64; count uint64 }{
		"fs_op=BatchForget": {val: 912, count: 2},
		"fs_op=CreateFile": {val: 456, count: 1},
	}for _, dp := range dataPoints {
		var parts []string
		dp.Attributes.Range(func(kv attribute.KeyValue) bool {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
			return true
		})
		sort.Strings(parts)
		key := strings.Join(parts, ";")

		e, ok := expected[key]
		assert.True(t, ok, "unexpected attribute set: %s", key)
		assert.Equal(t, e.count, dp.Count, "count mismatch for %s", key)
		assert.Equal(t, e.val, dp.Sum, "sum mismatch for %s", key)
		delete(expected, key)
	}
	assert.Empty(t, expected, "not all expected attribute sets were found")
}

func TestFsOpsErrorCount(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	
	
	m.FsOpsErrorCount(
		5,"DEVICE_ERROR", "BatchForget")m.FsOpsErrorCount(
		2,"DEVICE_ERROR", "CreateFile")m.FsOpsErrorCount(
		3,"DEVICE_ERROR", "BatchForget")

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "fs/ops_error_count" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric fs/ops_error_count not found")
	assert.Equal(t, "The cumulative number of errors generated by file system operations.", foundMetric.Description)

	sum, ok := foundMetric.Data.(metricdata.Sum[int64])
	assert.True(t, ok, "metric fs/ops_error_count is not a Sum")
	dataPoints := sum.DataPoints

	assert.Len(t, dataPoints, 2)
	expected := map[string]struct{ val int64; count uint64 }{
		"fs_error_category=DEVICE_ERROR;fs_op=BatchForget": {val: 8, count: 2},
		"fs_error_category=DEVICE_ERROR;fs_op=CreateFile": {val: 2, count: 1},
	}for _, dp := range dataPoints {
		var parts []string
		dp.Attributes.Range(func(kv attribute.KeyValue) bool {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
			return true
		})
		sort.Strings(parts)
		key := strings.Join(parts, ";")

		e, ok := expected[key]
		assert.True(t, ok, "unexpected attribute set: %s", key)
		assert.Equal(t, e.val, dp.Value, "value mismatch for %s", key)
		delete(expected, key)
	}
	assert.Empty(t, expected, "not all expected attribute sets were found")
}

func TestGcsReadCount(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	
	
	m.GcsReadCount(
		5,"Parallel")m.GcsReadCount(
		2,"Random")m.GcsReadCount(
		3,"Parallel")

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "gcs/read_count" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric gcs/read_count not found")
	assert.Equal(t, "Specifies the number of gcs reads made along with type - Sequential/Random", foundMetric.Description)

	sum, ok := foundMetric.Data.(metricdata.Sum[int64])
	assert.True(t, ok, "metric gcs/read_count is not a Sum")
	dataPoints := sum.DataPoints

	assert.Len(t, dataPoints, 2)
	expected := map[string]struct{ val int64; count uint64 }{
		"read_type=Parallel": {val: 8, count: 2},
		"read_type=Random": {val: 2, count: 1},
	}for _, dp := range dataPoints {
		var parts []string
		dp.Attributes.Range(func(kv attribute.KeyValue) bool {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
			return true
		})
		sort.Strings(parts)
		key := strings.Join(parts, ";")

		e, ok := expected[key]
		assert.True(t, ok, "unexpected attribute set: %s", key)
		assert.Equal(t, e.val, dp.Value, "value mismatch for %s", key)
		delete(expected, key)
	}
	assert.Empty(t, expected, "not all expected attribute sets were found")
}

func TestGcsDownloadBytesCount(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	
	
	m.GcsDownloadBytesCount(
		5,"Parallel")m.GcsDownloadBytesCount(
		2,"Random")m.GcsDownloadBytesCount(
		3,"Parallel")

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "gcs/download_bytes_count" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric gcs/download_bytes_count not found")
	assert.Equal(t, "The cumulative number of bytes downloaded from GCS along with type - Sequential/Random", foundMetric.Description)

	sum, ok := foundMetric.Data.(metricdata.Sum[int64])
	assert.True(t, ok, "metric gcs/download_bytes_count is not a Sum")
	dataPoints := sum.DataPoints

	assert.Len(t, dataPoints, 2)
	expected := map[string]struct{ val int64; count uint64 }{
		"read_type=Parallel": {val: 8, count: 2},
		"read_type=Random": {val: 2, count: 1},
	}for _, dp := range dataPoints {
		var parts []string
		dp.Attributes.Range(func(kv attribute.KeyValue) bool {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
			return true
		})
		sort.Strings(parts)
		key := strings.Join(parts, ";")

		e, ok := expected[key]
		assert.True(t, ok, "unexpected attribute set: %s", key)
		assert.Equal(t, e.val, dp.Value, "value mismatch for %s", key)
		delete(expected, key)
	}
	assert.Empty(t, expected, "not all expected attribute sets were found")
}

func TestGcsReadBytesCount(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	
	
	m.GcsReadBytesCount(5)

	m.GcsReadBytesCount(3)

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "gcs/read_bytes_count" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric gcs/read_bytes_count not found")
	assert.Equal(t, "The cumulative number of bytes read from GCS objects.", foundMetric.Description)

	sum, ok := foundMetric.Data.(metricdata.Sum[int64])
	assert.True(t, ok, "metric gcs/read_bytes_count is not a Sum")
	dataPoints := sum.DataPoints

	assert.Len(t, dataPoints, 1)
	dp := dataPoints[0]
	assert.Equal(t, int64(8), dp.Value)
}

func TestGcsReaderCount(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	
	
	m.GcsReaderCount(
		5,"closed")m.GcsReaderCount(
		2,"opened")m.GcsReaderCount(
		3,"closed")

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "gcs/reader_count" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric gcs/reader_count not found")
	assert.Equal(t, "The cumulative number of GCS object readers opened or closed.", foundMetric.Description)

	sum, ok := foundMetric.Data.(metricdata.Sum[int64])
	assert.True(t, ok, "metric gcs/reader_count is not a Sum")
	dataPoints := sum.DataPoints

	assert.Len(t, dataPoints, 2)
	expected := map[string]struct{ val int64; count uint64 }{
		"io_method=closed": {val: 8, count: 2},
		"io_method=opened": {val: 2, count: 1},
	}for _, dp := range dataPoints {
		var parts []string
		dp.Attributes.Range(func(kv attribute.KeyValue) bool {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
			return true
		})
		sort.Strings(parts)
		key := strings.Join(parts, ";")

		e, ok := expected[key]
		assert.True(t, ok, "unexpected attribute set: %s", key)
		assert.Equal(t, e.val, dp.Value, "value mismatch for %s", key)
		delete(expected, key)
	}
	assert.Empty(t, expected, "not all expected attribute sets were found")
}

func TestGcsRequestCount(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	
	
	m.GcsRequestCount(
		5,"ComposeObjects")m.GcsRequestCount(
		2,"CreateFolder")m.GcsRequestCount(
		3,"ComposeObjects")

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "gcs/request_count" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric gcs/request_count not found")
	assert.Equal(t, "The cumulative number of GCS requests processed along with the GCS method.", foundMetric.Description)

	sum, ok := foundMetric.Data.(metricdata.Sum[int64])
	assert.True(t, ok, "metric gcs/request_count is not a Sum")
	dataPoints := sum.DataPoints

	assert.Len(t, dataPoints, 2)
	expected := map[string]struct{ val int64; count uint64 }{
		"gcs_method=ComposeObjects": {val: 8, count: 2},
		"gcs_method=CreateFolder": {val: 2, count: 1},
	}for _, dp := range dataPoints {
		var parts []string
		dp.Attributes.Range(func(kv attribute.KeyValue) bool {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
			return true
		})
		sort.Strings(parts)
		key := strings.Join(parts, ";")

		e, ok := expected[key]
		assert.True(t, ok, "unexpected attribute set: %s", key)
		assert.Equal(t, e.val, dp.Value, "value mismatch for %s", key)
		delete(expected, key)
	}
	assert.Empty(t, expected, "not all expected attribute sets were found")
}

func TestGcsRequestLatencies(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	
	
	m.GcsRequestLatencies(
		ctx, 123*time.Millisecond,"ComposeObjects")m.GcsRequestLatencies(
		ctx, 456*time.Millisecond,"CreateFolder")m.GcsRequestLatencies(
		ctx, 789*time.Millisecond,"ComposeObjects")

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "gcs/request_latencies" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric gcs/request_latencies not found")
	assert.Equal(t, "The cumulative distribution of the GCS request latencies.", foundMetric.Description)

	hist, ok := foundMetric.Data.(metricdata.Histogram[int64])
	assert.True(t, ok, "metric gcs/request_latencies is not a Histogram")
	dataPoints := hist.DataPoints

	assert.Len(t, dataPoints, 2)
	expected := map[string]struct{ val int64; count uint64 }{
		"gcs_method=ComposeObjects": {val: 912, count: 2},
		"gcs_method=CreateFolder": {val: 456, count: 1},
	}for _, dp := range dataPoints {
		var parts []string
		dp.Attributes.Range(func(kv attribute.KeyValue) bool {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
			return true
		})
		sort.Strings(parts)
		key := strings.Join(parts, ";")

		e, ok := expected[key]
		assert.True(t, ok, "unexpected attribute set: %s", key)
		assert.Equal(t, e.count, dp.Count, "count mismatch for %s", key)
		assert.Equal(t, e.val, dp.Sum, "sum mismatch for %s", key)
		delete(expected, key)
	}
	assert.Empty(t, expected, "not all expected attribute sets were found")
}

func TestGcsRetryCount(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	
	
	m.GcsRetryCount(
		5,"OTHER_ERRORS")m.GcsRetryCount(
		2,"STALLED_READ_REQUEST")m.GcsRetryCount(
		3,"OTHER_ERRORS")

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "gcs/retry_count" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric gcs/retry_count not found")
	assert.Equal(t, "The cumulative number of retry requests made to GCS.", foundMetric.Description)

	sum, ok := foundMetric.Data.(metricdata.Sum[int64])
	assert.True(t, ok, "metric gcs/retry_count is not a Sum")
	dataPoints := sum.DataPoints

	assert.Len(t, dataPoints, 2)
	expected := map[string]struct{ val int64; count uint64 }{
		"retry_error_category=OTHER_ERRORS": {val: 8, count: 2},
		"retry_error_category=STALLED_READ_REQUEST": {val: 2, count: 1},
	}for _, dp := range dataPoints {
		var parts []string
		dp.Attributes.Range(func(kv attribute.KeyValue) bool {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
			return true
		})
		sort.Strings(parts)
		key := strings.Join(parts, ";")

		e, ok := expected[key]
		assert.True(t, ok, "unexpected attribute set: %s", key)
		assert.Equal(t, e.val, dp.Value, "value mismatch for %s", key)
		delete(expected, key)
	}
	assert.Empty(t, expected, "not all expected attribute sets were found")
}

func TestFileCacheReadCount(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	
	
	m.FileCacheReadCount(
		5,true, "Parallel")m.FileCacheReadCount(
		2,true, "Random")m.FileCacheReadCount(
		3,true, "Parallel")

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "file_cache/read_count" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric file_cache/read_count not found")
	assert.Equal(t, "Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false", foundMetric.Description)

	sum, ok := foundMetric.Data.(metricdata.Sum[int64])
	assert.True(t, ok, "metric file_cache/read_count is not a Sum")
	dataPoints := sum.DataPoints

	assert.Len(t, dataPoints, 2)
	expected := map[string]struct{ val int64; count uint64 }{
		"cache_hit=true;read_type=Parallel": {val: 8, count: 2},
		"cache_hit=true;read_type=Random": {val: 2, count: 1},
	}for _, dp := range dataPoints {
		var parts []string
		dp.Attributes.Range(func(kv attribute.KeyValue) bool {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
			return true
		})
		sort.Strings(parts)
		key := strings.Join(parts, ";")

		e, ok := expected[key]
		assert.True(t, ok, "unexpected attribute set: %s", key)
		assert.Equal(t, e.val, dp.Value, "value mismatch for %s", key)
		delete(expected, key)
	}
	assert.Empty(t, expected, "not all expected attribute sets were found")
}

func TestFileCacheReadBytesCount(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	
	
	m.FileCacheReadBytesCount(
		5,"Parallel")m.FileCacheReadBytesCount(
		2,"Random")m.FileCacheReadBytesCount(
		3,"Parallel")

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "file_cache/read_bytes_count" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric file_cache/read_bytes_count not found")
	assert.Equal(t, "The cumulative number of bytes read from file cache along with read type - Sequential/Random", foundMetric.Description)

	sum, ok := foundMetric.Data.(metricdata.Sum[int64])
	assert.True(t, ok, "metric file_cache/read_bytes_count is not a Sum")
	dataPoints := sum.DataPoints

	assert.Len(t, dataPoints, 2)
	expected := map[string]struct{ val int64; count uint64 }{
		"read_type=Parallel": {val: 8, count: 2},
		"read_type=Random": {val: 2, count: 1},
	}for _, dp := range dataPoints {
		var parts []string
		dp.Attributes.Range(func(kv attribute.KeyValue) bool {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
			return true
		})
		sort.Strings(parts)
		key := strings.Join(parts, ";")

		e, ok := expected[key]
		assert.True(t, ok, "unexpected attribute set: %s", key)
		assert.Equal(t, e.val, dp.Value, "value mismatch for %s", key)
		delete(expected, key)
	}
	assert.Empty(t, expected, "not all expected attribute sets were found")
}

func TestFileCacheReadLatencies(t *testing.T) {
	// Arrange
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	ctx := context.Background()
	m, err := NewOTelMetrics(ctx, 1, 100)
	assert.NoError(t, err)

	// Act
	
	
	m.FileCacheReadLatencies(
		ctx, 123*time.Microsecond,true)m.FileCacheReadLatencies(
		ctx, 456*time.Microsecond,false)m.FileCacheReadLatencies(
		ctx, 789*time.Microsecond,true)

	waitForMetrics()

	// Assert
	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)
	assert.Len(t, rm.ScopeMetrics, 1, "expected 1 scope metric")
	sm := rm.ScopeMetrics[0]

	var foundMetric metricdata.Metrics
	for _, met := range sm.Metrics {
		if met.Name == "file_cache/read_latencies" {
			foundMetric = met
			break
		}
	}
	assert.NotNil(t, foundMetric, "metric file_cache/read_latencies not found")
	assert.Equal(t, "The cumulative distribution of the file cache read latencies along with cache hit - true/false.", foundMetric.Description)

	hist, ok := foundMetric.Data.(metricdata.Histogram[int64])
	assert.True(t, ok, "metric file_cache/read_latencies is not a Histogram")
	dataPoints := hist.DataPoints

	assert.Len(t, dataPoints, 2)
	expected := map[string]struct{ val int64; count uint64 }{
		"cache_hit=true": {val: 912, count: 2},
		"cache_hit=false": {val: 456, count: 1},
	}for _, dp := range dataPoints {
		var parts []string
		dp.Attributes.Range(func(kv attribute.KeyValue) bool {
			parts = append(parts, fmt.Sprintf("%s=%s", kv.Key, kv.Value.AsString()))
			return true
		})
		sort.Strings(parts)
		key := strings.Join(parts, ";")

		e, ok := expected[key]
		assert.True(t, ok, "unexpected attribute set: %s", key)
		assert.Equal(t, e.count, dp.Count, "count mismatch for %s", key)
		assert.Equal(t, e.val, dp.Sum, "sum mismatch for %s", key)
		delete(expected, key)
	}
	assert.Empty(t, expected, "not all expected attribute sets were found")
}

