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

package optimizedmetrics

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	fileCacheReadBytesCountReadTypeParallelAttrSet   = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Parallel")))
	fileCacheReadBytesCountReadTypeRandomAttrSet     = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Random")))
	fileCacheReadBytesCountReadTypeSequentialAttrSet = metric.WithAttributeSet(attribute.NewSet(attribute.String("read_type", "Sequential")))
	fileCacheReadLatenciesCacheHitTrueAttrSet        = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", true)))
	fileCacheReadLatenciesCacheHitFalseAttrSet       = metric.WithAttributeSet(attribute.NewSet(attribute.Bool("cache_hit", false)))
)

type histogramRecord struct {
	ctx        context.Context
	instrument metric.Int64Histogram
	value      int64
	attributes metric.RecordOption
}

type otelMetrics struct {
	ch                                              chan histogramRecord
	fileCacheReadBytesCountReadTypeParallelAtomic   *atomic.Int64
	fileCacheReadBytesCountReadTypeRandomAtomic     *atomic.Int64
	fileCacheReadBytesCountReadTypeSequentialAtomic *atomic.Int64
	fileCacheReadLatencies                          metric.Int64Histogram
}

func (o *otelMetrics) FileCacheReadBytesCount(
	inc int64, readType string,
) {
	switch readType {
	case "Parallel":
		o.fileCacheReadBytesCountReadTypeParallelAtomic.Add(inc)
	case "Random":
		o.fileCacheReadBytesCountReadTypeRandomAtomic.Add(inc)
	case "Sequential":
		o.fileCacheReadBytesCountReadTypeSequentialAtomic.Add(inc)
	}

}

func (o *otelMetrics) FileCacheReadLatencies(
	ctx context.Context, latency time.Duration, cacheHit bool,
) {
	var record histogramRecord
	switch cacheHit {
	case true:
		record = histogramRecord{ctx: ctx, instrument: o.fileCacheReadLatencies, value: latency.Microseconds(), attributes: fileCacheReadLatenciesCacheHitTrueAttrSet}
	case false:
		record = histogramRecord{ctx: ctx, instrument: o.fileCacheReadLatencies, value: latency.Microseconds(), attributes: fileCacheReadLatenciesCacheHitFalseAttrSet}
	}

	select {
	case o.ch <- histogramRecord{instrument: record.instrument, value: record.value, attributes: record.attributes, ctx: ctx}: // Do nothing
	default: // Unblock writes to channel if it's full.
	}
}

func NewOTelMetrics(ctx context.Context, workers int, bufferSize int) (*otelMetrics, error) {
	ch := make(chan histogramRecord, bufferSize)
	for range workers {
		go func() {
			for record := range ch {
				record.instrument.Record(record.ctx, record.value, record.attributes)
			}
		}()
	}
	meter := otel.Meter("gcsfuse")
	var fileCacheReadBytesCountReadTypeParallelAtomic,
		fileCacheReadBytesCountReadTypeRandomAtomic,
		fileCacheReadBytesCountReadTypeSequentialAtomic atomic.Int64

	_, err0 := meter.Int64ObservableCounter("file_cache/read_bytes_count",
		metric.WithDescription("The cumulative number of bytes read from file cache along with read type - Sequential/Random"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(fileCacheReadBytesCountReadTypeParallelAtomic.Load(), fileCacheReadBytesCountReadTypeParallelAttrSet)
			obsrv.Observe(fileCacheReadBytesCountReadTypeRandomAtomic.Load(), fileCacheReadBytesCountReadTypeRandomAttrSet)
			obsrv.Observe(fileCacheReadBytesCountReadTypeSequentialAtomic.Load(), fileCacheReadBytesCountReadTypeSequentialAttrSet)
			return nil
		}))

	fileCacheReadLatencies, err1 := meter.Int64Histogram("file_cache/read_latencies",
		metric.WithDescription("The cumulative distribution of the file cache read latencies along with cache hit - true/false."),
		metric.WithUnit("us"),
		metric.WithExplicitBucketBoundaries(1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000))

	errs := []error{err0, err1}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	return &otelMetrics{
		ch: ch,
		fileCacheReadBytesCountReadTypeParallelAtomic:   &fileCacheReadBytesCountReadTypeParallelAtomic,
		fileCacheReadBytesCountReadTypeRandomAtomic:     &fileCacheReadBytesCountReadTypeRandomAtomic,
		fileCacheReadBytesCountReadTypeSequentialAtomic: &fileCacheReadBytesCountReadTypeSequentialAtomic,
		fileCacheReadLatencies:                          fileCacheReadLatencies,
	}, nil
}

func (o *otelMetrics) Close() {
	close(o.ch)
}
