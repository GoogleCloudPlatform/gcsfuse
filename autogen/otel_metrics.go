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
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/otel/attribute"
)

var (
	meter = otel.Meter("gcsfuse")
)

type otelMetrics struct {
	metric1Attr1V1Atomic, metric1Attr1V2Atomic, metric2Attr2Op1Attr3TrueAtomic, metric2Attr2Op1Attr3FalseAtomic, metric2Attr2Op2Attr3TrueAtomic, metric2Op2Attr3FalseAtomic *atomic.Int64
}

func NewOTelMetrics() (*otelMetrics, error) {
	var metric1Attr1V1Atomic, metric1Attr1V2Atomic, metric2Attr2Op1Attr3TrueAtomic, metric2Attr2Op1Attr3FalseAtomic, metric2Attr2Op2Attr3TrueAtomic, metric2Op2Attr3FalseAtomic atomic.Int64

	metric1Attr1V1AttrSet := metric.WithAttributeSet(attribute.NewSet(attribute.String("attr1", "v1")))
	metric1Attr1V2AttrSet := metric.WithAttributeSet(attribute.NewSet(attribute.String("attr1", "v2")))
	metric2Attr2Op1Attr3TrueAttrSet := metric.WithAttributeSet(attribute.NewSet(attribute.String("attr2", "op1"), attribute.Bool("attr3", true)))
	metric2Attr2Op1Attr3FalseAttrSet := metric.WithAttributeSet(attribute.NewSet(attribute.String("attr2", "op1"), attribute.Bool("attr3", false)))
	metric2Attr2Op2Attr3TrueAttrSet := metric.WithAttributeSet(attribute.NewSet(attribute.String("attr2", "op2"), attribute.Bool("attr3", true)))
	metric2Attr2Op2Attr3FalseAttrSet := metric.WithAttributeSet(attribute.NewSet(attribute.String("attr2", "op2"), attribute.Bool("attr3", false)))
	meter.Int64ObservableCounter("metric1",
		metric.WithDescription("description of the metric1"),
		metric.WithUnit("us"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(metric1Attr1V1Atomic.Load(), metric1Attr1V1AttrSet)
			obsrv.Observe(metric1Attr1V2Atomic.Load(), metric1Attr1V2AttrSet)
			return nil
		}))

	meter.Int64ObservableCounter("metric2",
		metric.WithDescription("description of the metric2"),
		metric.WithUnit("by"),
		metric.WithInt64Callback(func(_ context.Context, obsrv metric.Int64Observer) error {
			obsrv.Observe(metric2Attr2Op1Attr3TrueAtomic.Load(), metric2Attr2Op1Attr3TrueAttrSet)
			obsrv.Observe(metric2Attr2Op1Attr3FalseAtomic.Load(), metric2Attr2Op1Attr3FalseAttrSet)
			obsrv.Observe(metric2Attr2Op2Attr3TrueAtomic.Load(), metric2Attr2Op2Attr3TrueAttrSet)
			obsrv.Observe(metric2Op2Attr3FalseAtomic.Load(), metric2Attr2Op2Attr3FalseAttrSet)
			return nil
		}))

	return &otelMetrics{
		metric1Attr1V1Atomic:            &metric1Attr1V1Atomic,
		metric1Attr1V2Atomic:            &metric1Attr1V2Atomic,
		metric2Attr2Op1Attr3TrueAtomic:  &metric2Attr2Op1Attr3TrueAtomic,
		metric2Attr2Op1Attr3FalseAtomic: &metric2Attr2Op1Attr3FalseAtomic,
		metric2Attr2Op2Attr3TrueAtomic:  &metric2Attr2Op2Attr3TrueAtomic,
		metric2Op2Attr3FalseAtomic:      &metric2Op2Attr3FalseAtomic,
	}, nil
}
