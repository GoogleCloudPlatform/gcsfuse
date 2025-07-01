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

package main

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/otel/attribute"
)

var (
	meter                            = otel.Meter("gcsfuse")
	metric1Attr1V1AttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("attr1", "v1")))
	metric1Attr1V2AttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("attr1", "v2")))
	metric2Attr2Op1Attr3TrueAttrSet  = metric.WithAttributeSet(attribute.NewSet(attribute.String("attr2", "op1"), attribute.Bool("attr3", true)))
	metric2Attr2Op1Attr3FalseAttrSet = metric.WithAttributeSet(attribute.NewSet(attribute.String("attr2", "op1"), attribute.Bool("attr3", false)))
	metric2Attr2Op2Attr3TrueAttrSet  = metric.WithAttributeSet(attribute.NewSet(attribute.String("attr2", "op2"), attribute.Bool("attr3", true)))
	metric2Attr2Op2Attr3FalseAttrSet = metric.WithAttributeSet(attribute.NewSet(attribute.String("attr2", "op2"), attribute.Bool("attr3", false)))
	metric3Attr4A1AttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("attr4", "a1")))
	metric3Attr4A2AttrSet            = metric.WithAttributeSet(attribute.NewSet(attribute.String("attr4", "a2")))
)

type MetricHandle interface {
	Metric1(inc int64, attr1 string)
	Metric2(inc int64, attr2 string, attr3 bool)
	Metric3(ctx context.Context, duration time.Duration, attr4 string)
	Metric4(ctx context.Context, duration time.Duration)
}

type otelMetrics struct {
	metric1Attr1V1Atomic,
	metric1Attr1V2Atomic,
	metric2Attr2Op1Attr3TrueAtomic,
	metric2Attr2Op1Attr3FalseAtomic,
	metric2Attr2Op2Attr3TrueAtomic,
	metric2Attr2Op2Attr3FalseAtomic *atomic.Int64
	metric3,
	metric4 metric.Int64Histogram
}

func (o *otelMetrics) Metric1(inc int64, attr1 string) {
	switch attr1 {
	case "v1":
		o.metric1Attr1V1Atomic.Add(inc)
	case "v2":
		o.metric1Attr1V2Atomic.Add(inc)
	}
}

func (o *otelMetrics) Metric2(inc int64, attr2 string, attr3 bool) {
	switch attr2 {
	case "op1":
		switch attr3 {
		case true:
			o.metric2Attr2Op1Attr3TrueAtomic.Add(inc)
		case false:
			o.metric2Attr2Op1Attr3FalseAtomic.Add(inc)
		}
	case "op2":
		switch attr3 {
		case true:
			o.metric2Attr2Op2Attr3TrueAtomic.Add(inc)
		case false:
			o.metric2Attr2Op2Attr3FalseAtomic.Add(inc)
		}
	}
}

func (o *otelMetrics) Metric3(ctx context.Context, latency time.Duration, attr4 string) {
	switch attr4 {
	case "a1":
		o.metric3.Record(ctx, latency.Microseconds(), metric3Attr4A1AttrSet)
	case "a2":
		o.metric3.Record(ctx, latency.Microseconds(), metric3Attr4A2AttrSet)
	}
}

func (o *otelMetrics) Metric4(ctx context.Context, latency time.Duration) {
	o.metric4.Record(ctx, latency.Milliseconds())
}

func NewOTelMetrics() (*otelMetrics, error) {
	var metric1Attr1V1Atomic, metric1Attr1V2Atomic, metric2Attr2Op1Attr3TrueAtomic, metric2Attr2Op1Attr3FalseAtomic, metric2Attr2Op2Attr3TrueAtomic, metric2Attr2Op2Attr3FalseAtomic atomic.Int64

	_, err1 := meter.Int64ObservableCounter("metric1",
		metric.WithDescription("description of the metric1"),
		metric.WithUnit("us"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(metric1Attr1V1Atomic.Load(), metric1Attr1V1AttrSet)
			obsrv.Observe(metric1Attr1V2Atomic.Load(), metric1Attr1V2AttrSet)
			return nil
		}))

	_, err2 := meter.Int64ObservableCounter("metric2",
		metric.WithDescription("description of the metric2"),
		metric.WithUnit("by"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(metric2Attr2Op1Attr3TrueAtomic.Load(), metric2Attr2Op1Attr3TrueAttrSet)
			obsrv.Observe(metric2Attr2Op1Attr3FalseAtomic.Load(), metric2Attr2Op1Attr3FalseAttrSet)
			obsrv.Observe(metric2Attr2Op2Attr3TrueAtomic.Load(), metric2Attr2Op2Attr3TrueAttrSet)
			obsrv.Observe(metric2Attr2Op2Attr3FalseAtomic.Load(), metric2Attr2Op2Attr3FalseAttrSet)
			return nil
		}))

	metric3, err3 := meter.Int64Histogram("metric3",
		metric.WithDescription("description of the metric3"),
		metric.WithUnit("us"),
		metric.WithExplicitBucketBoundaries(1, 160, 5000))

	metric4, err4 := meter.Int64Histogram("metric4",
		metric.WithDescription("description of the metric4"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(1, 150, 500))

	if err := errors.Join(err1, err2, err3, err4); err != nil {
		return nil, err
	}
	return &otelMetrics{
		metric1Attr1V1Atomic:            &metric1Attr1V1Atomic,
		metric1Attr1V2Atomic:            &metric1Attr1V2Atomic,
		metric2Attr2Op1Attr3TrueAtomic:  &metric2Attr2Op1Attr3TrueAtomic,
		metric2Attr2Op1Attr3FalseAtomic: &metric2Attr2Op1Attr3FalseAtomic,
		metric2Attr2Op2Attr3TrueAtomic:  &metric2Attr2Op2Attr3TrueAtomic,
		metric2Attr2Op2Attr3FalseAtomic: &metric2Attr2Op2Attr3FalseAtomic,
		metric3:                         metric3,
		metric4:                         metric4,
	}, nil
}
