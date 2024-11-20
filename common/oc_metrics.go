// Copyright 2024 Google LLC
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
	"sync"

	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	ocMetric    *ocMetrics
	ocInitError error
)

const (
	// IOMethod annotates the event that opens or closes a connection or file.
	IOMethod = "io_method"

	// GCSMethod annotates the method called in the GCS client library.
	GCSMethod = "gcs_method"

	// FSOp annotates the file system op processed.
	FSOp = "fs_op"

	// FSErrCategory reduces the cardinality of FSError by grouping errors together.
	FSErrCategory = "fs_error_category"

	// ReadType annotates the read operation with the type - Sequential/Random
	ReadType = "read_type"

	// CacheHit annotates the read operation from file cache with true or false.
	CacheHit = "cache_hit"
)

var ocOnce sync.Once

type ocMetrics struct {
	// GCS measures
	gcsReadBytesCount     *stats.Int64Measure
	gcsReaderCount        *stats.Int64Measure
	gcsRequestCount       *stats.Int64Measure
	gcsRequestLatency     *stats.Float64Measure
	gcsReadCount          *stats.Int64Measure
	gcsDownloadBytesCount *stats.Int64Measure

	// Ops measures
	opsCount      *stats.Int64Measure
	opsErrorCount *stats.Int64Measure
	opsLatency    *stats.Float64Measure

	// File cache measures
	fileCacheReadCount      *stats.Int64Measure
	fileCacheReadBytesCount *stats.Int64Measure
	fileCacheReadLatency    *stats.Float64Measure
}

func attrsToTags(attrs []Attr) []tag.Mutator {
	mutators := make([]tag.Mutator, 0, len(attrs))
	for _, attr := range attrs {
		mutators = append(mutators, tag.Upsert(tag.MustNewKey(attr.Key), attr.Value))
	}
	return mutators
}
func (o *ocMetrics) GCSReadBytesCount(ctx context.Context, inc int64, attrs []Attr) {
	recordOCMetric(ctx, o.gcsReadBytesCount, inc, attrs, "GCS read bytes count")
}

func (o *ocMetrics) GCSReaderCount(ctx context.Context, inc int64, attrs []Attr) {
	recordOCMetric(ctx, o.gcsReaderCount, inc, attrs, "GCS reader count")
}

func (o *ocMetrics) GCSRequestCount(ctx context.Context, inc int64, attrs []Attr) {
	recordOCMetric(ctx, o.gcsRequestCount, inc, attrs, "GCS request count")
}

func (o *ocMetrics) GCSRequestLatency(ctx context.Context, value float64, attrs []Attr) {
	recordOCLatencyMetric(ctx, o.gcsRequestLatency, value, attrs, "GCS request latency")
}
func (o *ocMetrics) GCSReadCount(ctx context.Context, inc int64, attrs []Attr) {
	recordOCMetric(ctx, o.gcsReadCount, inc, attrs, "GCS read count")
}
func (o *ocMetrics) GCSDownloadBytesCount(ctx context.Context, inc int64, attrs []Attr) {
	recordOCMetric(ctx, o.gcsDownloadBytesCount, inc, attrs, "GCS download bytes count")
}

func (o *ocMetrics) OpsCount(ctx context.Context, inc int64, attrs []Attr) {
	recordOCMetric(ctx, o.opsCount, inc, attrs, "file system op count")
}
func (o *ocMetrics) OpsLatency(ctx context.Context, value float64, attrs []Attr) {
	recordOCLatencyMetric(ctx, o.opsLatency, value, attrs, "file system op latency")
}
func (o *ocMetrics) OpsErrorCount(ctx context.Context, inc int64, attrs []Attr) {
	recordOCMetric(ctx, o.opsErrorCount, inc, attrs, "file system op error count")
}

func (o *ocMetrics) FileCacheReadCount(ctx context.Context, inc int64, attrs []Attr) {
	recordOCMetric(ctx, o.fileCacheReadCount, inc, attrs, "file cache read count")
}
func (o *ocMetrics) FileCacheReadBytesCount(ctx context.Context, inc int64, attrs []Attr) {
	recordOCMetric(ctx, o.fileCacheReadBytesCount, inc, attrs, "file cache read bytes count")
}
func (o *ocMetrics) FileCacheReadLatency(ctx context.Context, value float64, attrs []Attr) {
	recordOCLatencyMetric(ctx, o.fileCacheReadLatency, value, attrs, "file cache read latency")
}

func recordOCMetric(ctx context.Context, m *stats.Int64Measure, inc int64, attrs []Attr, metricStr string) {
	if err := stats.RecordWithTags(
		ctx,
		attrsToTags(attrs),
		m.M(inc),
	); err != nil {
		logger.Errorf("Cannot record %s: %v: %v", metricStr, attrs, err)
	}
}

func recordOCLatencyMetric(ctx context.Context, m *stats.Float64Measure, inc float64, attrs []Attr, metricStr string) {
	if err := stats.RecordWithTags(
		ctx,
		attrsToTags(attrs),
		m.M(inc),
	); err != nil {
		logger.Errorf("Cannot record %s: %v: %v", metricStr, attrs, err)
	}
}

func NewOCMetrics() (MetricHandle, error) {
	ocOnce.Do(func() {
		ocMetric, ocInitError = initOCMetrics()
	})
	return ocMetric, ocInitError
}

func initOCMetrics() (*ocMetrics, error) {
	gcsReadBytesCount := stats.Int64("gcs/read_bytes_count", "The number of bytes read from GCS objects.", stats.UnitBytes)
	gcsReaderCount := stats.Int64("gcs/reader_count", "The number of GCS object readers opened or closed.", stats.UnitDimensionless)
	gcsRequestCount := stats.Int64("gcs/request_count", "The number of GCS requests processed.", stats.UnitDimensionless)
	gcsRequestLatency := stats.Float64("gcs/request_latency", "The latency of a GCS request.", stats.UnitMilliseconds)
	gcsReadCount := stats.Int64("gcs/read_count", "Specifies the number of gcs reads made along with type - Sequential/Random", stats.UnitDimensionless)
	gcsDownloadBytesCount := stats.Int64("gcs/download_bytes_count", "The cumulative number of bytes downloaded from GCS along with type - Sequential/Random", stats.UnitBytes)

	opsCount := stats.Int64("fs/ops_count", "The number of ops processed by the file system.", stats.UnitDimensionless)
	opsLatency := stats.Float64("fs/ops_latency", "The latency of a file system operation.", "us")
	opsErrorCount := stats.Int64("fs/ops_error_count", "The number of errors generated by file system operation.", stats.UnitDimensionless)

	fileCacheReadCount := stats.Int64("file_cache/read_count", "Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false", stats.UnitDimensionless)
	fileCacheReadBytesCount := stats.Int64("file_cache/read_bytes_count", "The cumulative number of bytes read from file cache along with read type - Sequential/Random", stats.UnitBytes)
	fileCacheReadLatency := stats.Float64("file_cache/read_latency", "Latency of read from file cache along with cache hit - true/false", "us")
	// OpenCensus views (aggregated measures)
	if err := view.Register(
		&view.View{
			Name:        "gcs/read_bytes_count",
			Measure:     gcsReadBytesCount,
			Description: "The cumulative number of bytes read from GCS objects.",
			Aggregation: view.Sum(),
		},
		&view.View{
			Name:        "gcs/reader_count",
			Measure:     gcsReaderCount,
			Description: "The cumulative number of GCS object readers opened or closed.",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tag.MustNewKey(IOMethod)},
		},
		&view.View{
			Name:        "gcs/request_count",
			Measure:     gcsRequestCount,
			Description: "The cumulative number of GCS requests processed.",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tag.MustNewKey(GCSMethod)},
		},
		&view.View{
			Name:        "gcs/request_latencies",
			Measure:     gcsRequestLatency,
			Description: "The cumulative distribution of the GCS request latencies.",
			Aggregation: ochttp.DefaultLatencyDistribution,
			TagKeys:     []tag.Key{tag.MustNewKey(GCSMethod)},
		},
		&view.View{
			Name:        "gcs/read_count",
			Measure:     gcsReadCount,
			Description: "Specifies the number of gcs reads made along with type - Sequential/Random",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tag.MustNewKey(ReadType)},
		},
		&view.View{
			Name:        "gcs/download_bytes_count",
			Measure:     gcsDownloadBytesCount,
			Description: "The cumulative number of bytes downloaded from GCS along with type - Sequential/Random",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tag.MustNewKey(ReadType)},
		},
		&view.View{
			Name:        "fs/ops_count",
			Measure:     opsCount,
			Description: "The cumulative number of ops processed by the file system.",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tag.MustNewKey(FSOp)},
		},
		&view.View{
			Name:        "fs/ops_error_count",
			Measure:     opsErrorCount,
			Description: "The cumulative number of errors generated by file system operations",
			Aggregation: view.Sum(),
			TagKeys:     []tag.Key{tag.MustNewKey(FSOp), tag.MustNewKey(FSErrCategory)},
		},
		&view.View{
			Name:        "fs/ops_latency",
			Measure:     opsLatency,
			Description: "The cumulative distribution of file system operation latencies",
			Aggregation: ochttp.DefaultLatencyDistribution,
			TagKeys:     []tag.Key{tag.MustNewKey(FSOp)},
		}); err != nil {
		return nil, fmt.Errorf("failed to register OpenCensus metrics for GCS client library: %w", err)
	}
	return &ocMetrics{
		gcsReadBytesCount:     gcsReadBytesCount,
		gcsReaderCount:        gcsReaderCount,
		gcsRequestCount:       gcsRequestCount,
		gcsRequestLatency:     gcsRequestLatency,
		gcsReadCount:          gcsReadCount,
		gcsDownloadBytesCount: gcsDownloadBytesCount,

		opsCount:      opsCount,
		opsErrorCount: opsErrorCount,
		opsLatency:    opsLatency,

		fileCacheReadCount:      fileCacheReadCount,
		fileCacheReadBytesCount: fileCacheReadBytesCount,
		fileCacheReadLatency:    fileCacheReadLatency,
	}, nil
}
