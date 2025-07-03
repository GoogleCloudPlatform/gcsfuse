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

var (
	fsOpsMeter     = otel.Meter("fs_op")
	gcsMeter       = otel.Meter("gcs")
	fileCacheMeter = otel.Meter("file_cache")

	// Attribute Keys
	// ioMethodKey specifies the I/O method attribute (e.g., opened, closed).
	ioMethodKey = attribute.Key("io_method")
	// gcsMethodKey specifies the name of the GCS method
	gcsMethodKey = attribute.Key("gcs_method")
	// fsOpKey specifies the FS operation like LookupInode, ReadFile etc.
	fsOpKey = attribute.Key("fs_op")
	// fsErrCategoryKey specifies the error category. The intention is to reduce the cardinality of FSError by grouping errors together.
	fsErrCategoryKey = attribute.Key("fs_error_category")
	// readTypeKey specifies the read operation like whether it's Sequential or Random
	readTypeKey = attribute.Key("read_type")
	// cacheHitKey specifies whether the read operation from file cache resulted in a cache-hit or miss.
	cacheHitKey = attribute.Key("cache_hit")
	// retryErrCategoryKey specifies the category of the error that triggered a retry.
	retryErrCategoryKey = attribute.Key("retry_error_category")

	fsOpsOptionCache,
	readTypeOptionCache,
	ioMethodOptionCache,
	gcsMethodOptionCache,
	retryErrCategoryOptionCache,
	cacheHitOptionCache,
	cacheHitReadTypeOptionCache,
	fsOpsErrorCategoryOptionCache sync.Map
)

func loadOrStoreAttrOption[K comparable](mp *sync.Map, key K, attrSetGenFunc func() attribute.Set) metric.MeasurementOption {
	attrSet, ok := mp.Load(key)
	if ok {
		return attrSet.(metric.MeasurementOption)
	}
	v, _ := mp.LoadOrStore(key, metric.WithAttributeSet(attrSetGenFunc()))
	return v.(metric.MeasurementOption)
}

func ioMethodAttrOption(ioMethod string) metric.MeasurementOption {
	return loadOrStoreAttrOption(&ioMethodOptionCache, ioMethod,
		func() attribute.Set {
			return attribute.NewSet(ioMethodKey.String(ioMethod))
		})
}

func readTypeAttrOption(readType string) metric.MeasurementOption {
	return loadOrStoreAttrOption(&readTypeOptionCache, readType,
		func() attribute.Set {
			return attribute.NewSet(readTypeKey.String(readType))
		})
}

func fsOpsAttrOption(fsOps string) metric.MeasurementOption {
	return loadOrStoreAttrOption(&fsOpsOptionCache, fsOps,
		func() attribute.Set {
			return attribute.NewSet(fsOpKey.String(fsOps))
		})
}

func getFsOpsErrorCategoryAttributeOption(attr FSOpsErrorCategory) metric.MeasurementOption {
	return loadOrStoreAttrOption(&fsOpsErrorCategoryOptionCache, attr,
		func() attribute.Set {
			return attribute.NewSet(fsOpKey.String(attr.FSOps), fsErrCategoryKey.String(attr.ErrorCategory))
		})
}

func cacheHitAttrOption(cacheHit bool) metric.MeasurementOption {
	return loadOrStoreAttrOption(&cacheHitOptionCache, cacheHit,
		func() attribute.Set {
			return attribute.NewSet(cacheHitKey.Bool(cacheHit))
		})
}

func gcsMethodAttrOption(gcsMethod string) metric.MeasurementOption {
	return loadOrStoreAttrOption(&gcsMethodOptionCache, gcsMethod,
		func() attribute.Set {
			return attribute.NewSet(gcsMethodKey.String(gcsMethod))
		})
}

func retryErrCategoryAttrOption(retryErrCategory string) metric.MeasurementOption {
	return loadOrStoreAttrOption(&retryErrCategoryOptionCache, retryErrCategory,
		func() attribute.Set {
			return attribute.NewSet(retryErrCategoryKey.String(retryErrCategory))
		})
}

func cacheHitReadTypeAttrOption(attr CacheHitReadType) metric.MeasurementOption {
	return loadOrStoreAttrOption(&cacheHitReadTypeOptionCache, attr, func() attribute.Set {
		return attribute.NewSet(cacheHitKey.Bool(attr.CacheHit), readTypeKey.String(attr.ReadType))
	})
}

// otelMetrics maintains the list of all metrics computed in GCSFuse.
type otelMetrics struct {
	fsOpsCount      metric.Int64Counter
	fsOpsErrorCount metric.Int64Counter
	fsOpsLatency    metric.Int64Histogram

	gcsReadCount            metric.Int64Counter
	gcsReadBytesCountAtomic *atomic.Int64
	gcsReaderCount          metric.Int64Counter
	gcsRequestCount         metric.Int64Counter
	gcsRequestLatency       metric.Int64Histogram
	gcsDownloadBytesCount   metric.Int64Counter
	gcsRetryCount           metric.Int64Counter

	fileCacheReadCount      metric.Int64Counter
	fileCacheReadBytesCount metric.Int64Counter
	fileCacheReadLatency    metric.Int64Histogram
}

func (o *otelMetrics) GCSReadBytesCount(_ context.Context, inc int64) {
	o.gcsReadBytesCountAtomic.Add(inc)
}

func (o *otelMetrics) GCSReaderCount(ctx context.Context, inc int64, ioMethod string) {
	o.gcsReaderCount.Add(ctx, inc, ioMethodAttrOption(ioMethod))
}

func (o *otelMetrics) GCSRequestCount(ctx context.Context, inc int64, gcsMethod string) {
	o.gcsRequestCount.Add(ctx, inc, gcsMethodAttrOption(gcsMethod))
}

func (o *otelMetrics) GCSRequestLatency(ctx context.Context, latency time.Duration, gcsMethod string) {
	o.gcsRequestLatency.Record(ctx, latency.Milliseconds(), gcsMethodAttrOption(gcsMethod))
}

func (o *otelMetrics) GCSReadCount(ctx context.Context, inc int64, readType string) {
	o.gcsReadCount.Add(ctx, inc, readTypeAttrOption(readType))
}

func (o *otelMetrics) GCSDownloadBytesCount(ctx context.Context, inc int64, readType string) {
	o.gcsDownloadBytesCount.Add(ctx, inc, readTypeAttrOption(readType))
}

func (o *otelMetrics) GCSRetryCount(ctx context.Context, inc int64, retryErrCategory string) {
	o.gcsRetryCount.Add(ctx, inc, retryErrCategoryAttrOption(retryErrCategory))
}

func (o *otelMetrics) OpsCount(ctx context.Context, inc int64, fsOp string) {
	o.fsOpsCount.Add(ctx, inc, fsOpsAttrOption(fsOp))
}

func (o *otelMetrics) OpsLatency(ctx context.Context, latency time.Duration, fsOp string) {
	o.fsOpsLatency.Record(ctx, latency.Microseconds(), fsOpsAttrOption(fsOp))
}

func (o *otelMetrics) OpsErrorCount(ctx context.Context, inc int64, attrs FSOpsErrorCategory) {
	o.fsOpsErrorCount.Add(ctx, inc, getFsOpsErrorCategoryAttributeOption(attrs))
}

func (o *otelMetrics) FileCacheReadCount(ctx context.Context, inc int64, attrs CacheHitReadType) {
	o.fileCacheReadCount.Add(ctx, inc, cacheHitReadTypeAttrOption(attrs))
}

func (o *otelMetrics) FileCacheReadBytesCount(ctx context.Context, inc int64, readType string) {
	o.fileCacheReadBytesCount.Add(ctx, inc, readTypeAttrOption(readType))
}

func (o *otelMetrics) FileCacheReadLatency(ctx context.Context, latency time.Duration, cacheHit bool) {
	o.fileCacheReadLatency.Record(ctx, latency.Microseconds(), cacheHitAttrOption(cacheHit))
}

func NewOTelMetrics() (MetricHandle, error) {
	fsOpsCount, err1 := fsOpsMeter.Int64Counter("fs/ops_count", metric.WithDescription("The cumulative number of ops processed by the file system."))
	fsOpsLatency, err2 := fsOpsMeter.Int64Histogram("fs/ops_latency", metric.WithDescription("The cumulative distribution of file system operation latencies"), metric.WithUnit("us"),
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
	gcsRequestLatency, err9 := gcsMeter.Int64Histogram("gcs/request_latencies", metric.WithDescription("The cumulative distribution of the GCS request latencies."), metric.WithUnit("ms"))
	gcsRetryCount, err10 := gcsMeter.Int64Counter("gcs/retry_count", metric.WithDescription("The cumulative number of retry requests made to GCS."))

	fileCacheReadCount, err11 := fileCacheMeter.Int64Counter("file_cache/read_count",
		metric.WithDescription("Specifies the number of read requests made via file cache along with type - Sequential/Random and cache hit - true/false"))
	fileCacheReadBytesCount, err12 := fileCacheMeter.Int64Counter("file_cache/read_bytes_count",
		metric.WithDescription("The cumulative number of bytes read from file cache along with read type - Sequential/Random"),
		metric.WithUnit("By"))
	fileCacheReadLatency, err13 := fileCacheMeter.Int64Histogram("file_cache/read_latencies",
		metric.WithDescription("The cumulative distribution of the file cache read latencies along with cache hit - true/false"),
		metric.WithUnit("us"),
		defaultLatencyDistribution)

	if err := errors.Join(err1, err2, err3, err4, err5, err6, err7, err8, err9, err10, err11, err12, err13); err != nil {
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
		gcsRetryCount:           gcsRetryCount,
		fileCacheReadCount:      fileCacheReadCount,
		fileCacheReadBytesCount: fileCacheReadBytesCount,
		fileCacheReadLatency:    fileCacheReadLatency,
	}, nil
}
