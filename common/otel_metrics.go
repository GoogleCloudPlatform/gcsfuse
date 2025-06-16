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
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	// ioMethodKey annotates the event that opens or closes a connection or file.
	ioMethodKey = "io_method"

	// gcsMethodKey annotates the method called in the GCS client library.
	gcsMethodKey = "gcs_method"

	// fsOpKey annotates the file system op processed.
	fsOpKey = "fs_op"

	// fsErrCategoryKey reduces the cardinality of FSError by grouping errors together.
	fsErrCategoryKey = "fs_error_category"

	// readTypeKey annotates the read operation with the type - Sequential/Random
	readTypeKey = "read_type"

	// cacheHitKey annotates the read operation from file cache with true or false.
	cacheHitKey = "cache_hit"
)

var (
	fsOpsMeter     = otel.Meter("fs_op")
	gcsMeter       = otel.Meter("gcs")
	fileCacheMeter = otel.Meter("file_cache")

	fsOpsAttributeSet,
	readTypeAttributeSet,
	ioMethodAttributeSet,
	gcsMethodAttributeSet,
	cacheHitAttributeSet,
	cacheHitReadTypeAttributeSet,
	fsOpsErrorCategoryAttributeSet sync.Map
)

func loadOrStoreAttributeOption[K comparable](mp *sync.Map, key K, attrSetGenFunc func() attribute.Set) metric.MeasurementOption {
	attrSet, ok := mp.Load(key)
	if ok {
		return attrSet.(metric.MeasurementOption)
	}
	v, _ := mp.LoadOrStore(key, metric.WithAttributeSet(attrSetGenFunc()))
	return v.(metric.MeasurementOption)
}

func getIOMethodAttributeOption(ioMethod string) metric.MeasurementOption {
	return loadOrStoreAttributeOption(&ioMethodAttributeSet, ioMethod,
		func() attribute.Set {
			return attribute.NewSet(attribute.String(ioMethodKey, ioMethod))
		})
}

func getReadTypeAttributeOption(readType string) metric.MeasurementOption {
	return loadOrStoreAttributeOption(&readTypeAttributeSet, readType,
		func() attribute.Set {
			return attribute.NewSet(attribute.String(readTypeKey, readType))
		})
}

func getFSOpsAttributeOption(fsOps string) metric.MeasurementOption {
	return loadOrStoreAttributeOption(&fsOpsAttributeSet, fsOps,
		func() attribute.Set {
			return attribute.NewSet(attribute.String(fsOpKey, fsOps))
		})
}

func getFsOpsErrorCategoryAttributeOption(attr FSOpsErrorCategory) metric.MeasurementOption {
	return loadOrStoreAttributeOption(&fsOpsErrorCategoryAttributeSet, attr,
		func() attribute.Set {
			return attribute.NewSet(attribute.String(fsOpKey, attr.FSOps), attribute.String(fsErrCategoryKey, attr.ErrorCategory))
		})
}

func getCacheHitAttributeOption(cacheHit string) metric.MeasurementOption {
	return loadOrStoreAttributeOption(&cacheHitAttributeSet, cacheHit,
		func() attribute.Set {
			return attribute.NewSet(attribute.String(cacheHitKey, cacheHit))
		})
}

func getGCSMethodAttributeOption(gcsMethod string) metric.MeasurementOption {
	return loadOrStoreAttributeOption(&gcsMethodAttributeSet, gcsMethod,
		func() attribute.Set {
			return attribute.NewSet(attribute.String(gcsMethodKey, gcsMethod))
		})
}

func getCacheHitReadTypeAttributeOption(attr CacheHitReadType) metric.MeasurementOption {
	return loadOrStoreAttributeOption(&cacheHitReadTypeAttributeSet, attr, func() attribute.Set {
		return attribute.NewSet(attribute.String(cacheHitKey, attr.CacheHit), attribute.String(readTypeKey, attr.ReadType))
	})
}

// otelMetrics maintains the list of all metrics computed in GCSFuse.
type otelMetrics struct {
	fsOpsCount      metric.Int64Counter
	fsOpsErrorCount metric.Int64Counter
	fsOpsLatency    metric.Float64Histogram

	gcsReadCount            metric.Int64Counter
	gcsReadBytesCountAtomic *atomic.Int64
	gcsReaderCount          metric.Int64Counter
	gcsRequestCount         metric.Int64Counter
	gcsRequestLatency       metric.Float64Histogram
	gcsDownloadBytesCount   metric.Int64Counter

	fileCacheReadCount      metric.Int64Counter
	fileCacheReadBytesCount metric.Int64Counter
	fileCacheReadLatency    metric.Float64Histogram
}

func (o *otelMetrics) GCSReadBytesCount(_ context.Context, inc int64) {
	o.gcsReadBytesCountAtomic.Add(inc)
}

func (o *otelMetrics) GCSReaderCount(ctx context.Context, inc int64, ioMethod string) {
	o.gcsReaderCount.Add(ctx, inc, getIOMethodAttributeOption(ioMethod))
}

func (o *otelMetrics) GCSRequestCount(ctx context.Context, inc int64, gcsMethod string) {
	o.gcsRequestCount.Add(ctx, inc, getGCSMethodAttributeOption(gcsMethod))
}

func (o *otelMetrics) GCSRequestLatency(ctx context.Context, latency time.Duration, gcsMethod string) {
	o.gcsRequestLatency.Record(ctx, float64(latency.Milliseconds()), getGCSMethodAttributeOption(gcsMethod))
}

func (o *otelMetrics) GCSReadCount(ctx context.Context, inc int64, readType string) {
	o.gcsReadCount.Add(ctx, inc, getReadTypeAttributeOption(readType))
}

func (o *otelMetrics) GCSDownloadBytesCount(ctx context.Context, inc int64, readType string) {
	o.gcsDownloadBytesCount.Add(ctx, inc, getReadTypeAttributeOption(readType))
}

func (o *otelMetrics) OpsCount(ctx context.Context, inc int64, fsOp string) {
	o.fsOpsCount.Add(ctx, inc, getFSOpsAttributeOption(fsOp))
}

func (o *otelMetrics) OpsLatency(ctx context.Context, latency time.Duration, fsOp string) {
	o.fsOpsLatency.Record(ctx, float64(latency.Microseconds()), getFSOpsAttributeOption(fsOp))
}

func (o *otelMetrics) OpsErrorCount(ctx context.Context, inc int64, attrs FSOpsErrorCategory) {
	o.fsOpsErrorCount.Add(ctx, inc, getFsOpsErrorCategoryAttributeOption(attrs))
}

func (o *otelMetrics) FileCacheReadCount(ctx context.Context, inc int64, attrs CacheHitReadType) {
	o.fileCacheReadCount.Add(ctx, inc, getCacheHitReadTypeAttributeOption(attrs))
}

func (o *otelMetrics) FileCacheReadBytesCount(ctx context.Context, inc int64, readType string) {
	o.fileCacheReadBytesCount.Add(ctx, inc, getReadTypeAttributeOption(readType))
}

func (o *otelMetrics) FileCacheReadLatency(ctx context.Context, latency time.Duration, cacheHit string) {
	o.fileCacheReadLatency.Record(ctx, float64(latency.Microseconds()), getCacheHitAttributeOption(cacheHit))
}

func NewOTelMetrics() (MetricHandle, error) {
	fsOpsCount, err1 := fsOpsMeter.Int64Counter("fs/ops_count", metric.WithDescription("The cumulative number of ops processed by the file system."))
	fsOpsLatency, err2 := fsOpsMeter.Float64Histogram("fs/ops_latency", metric.WithDescription("The cumulative distribution of file system operation latencies"), metric.WithUnit("us"),
		defaultLatencyDistribution)
	fsOpsErrorCount, err3 := fsOpsMeter.Int64Counter("fs/ops_error_count", metric.WithDescription("The cumulative number of errors generated by file system operations"))

	gcsReadCount, err4 := gcsMeter.Int64Counter("gcs/read_count", metric.WithDescription("Specifies the number of gcs reads made along with type - Sequential/Random"))

	gcsDownloadBytesCount, err5 := gcsMeter.Int64Counter("gcs/download_bytes_count",
		metric.WithDescription("The cumulative number of bytes downloaded from GCS along with type - Sequential/Random"),
		metric.WithUnit("By"))

	var gcsReadBytesCountAtomic atomic.Int64
	_, err6 := gcsMeter.Int64ObservableCounter("gcs/read_bytes_count",
		metric.WithDescription("The cumulative number of bytes read from GCS objects."),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(gcsReadBytesCountAtomic.Load())
			return nil
		}))
	gcsReaderCount, err7 := gcsMeter.Int64Counter("gcs/reader_count", metric.WithDescription("The cumulative number of GCS object readers opened or closed."))
	gcsRequestCount, err8 := gcsMeter.Int64Counter("gcs/request_count", metric.WithDescription("The cumulative number of GCS requests processed along with the GCS method."))
	gcsRequestLatency, err9 := gcsMeter.Float64Histogram("gcs/request_latencies", metric.WithDescription("The cumulative distribution of the GCS request latencies."), metric.WithUnit("ms"))

	fileCacheReadCount, err10 := fileCacheMeter.Int64Counter("file_cache/read_count",
		metric.WithDescription("Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false"))
	fileCacheReadBytesCount, err11 := fileCacheMeter.Int64Counter("file_cache/read_bytes_count",
		metric.WithDescription("The cumulative number of bytes read from file cache along with read type - Sequential/Random"),
		metric.WithUnit("By"))
	fileCacheReadLatency, err12 := fileCacheMeter.Float64Histogram("file_cache/read_latencies",
		metric.WithDescription("The cumulative distribution of the file cache read latencies along with cache hit - true/false"),
		metric.WithUnit("us"),
		defaultLatencyDistribution)

	if err := errors.Join(err1, err2, err3, err4, err5, err6, err7, err8, err9, err10, err11, err12); err != nil {
		return nil, err
	}

	return &otelMetrics{
		fsOpsCount:              fsOpsCount,
		fsOpsErrorCount:         fsOpsErrorCount,
		fsOpsLatency:            fsOpsLatency,
		gcsReadCount:            gcsReadCount,
		gcsReadBytesCountAtomic: &gcsReadBytesCountAtomic,
		gcsReaderCount:          gcsReaderCount,
		gcsRequestCount:         gcsRequestCount,
		gcsRequestLatency:       gcsRequestLatency,
		gcsDownloadBytesCount:   gcsDownloadBytesCount,
		fileCacheReadCount:      fileCacheReadCount,
		fileCacheReadBytesCount: fileCacheReadBytesCount,
		fileCacheReadLatency:    fileCacheReadLatency,
	}, nil
}
