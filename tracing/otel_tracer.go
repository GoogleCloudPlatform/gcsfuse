// Copyright 2026 Google LLC
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

package tracing

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type otelTracer struct{}

func (o *otelTracer) StartTrace(ctx context.Context, traceName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := GCSFuseTracer.Start(ctx, traceName)
	span.SetAttributes(attrs...)
	return ctx, span
}

func (o *otelTracer) StartTraceLink(ctx context.Context, traceName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	span := trace.SpanFromContext(ctx)
	traceOpts := make([]trace.SpanStartOption, 0, 1)
	traceOpts = append(traceOpts, trace.WithLinks(trace.Link{
		SpanContext: span.SpanContext(),
		Attributes: []attribute.KeyValue{
			attribute.Int64("gcp.cloud_trace.link_type", 1),
		},
	}))
	ctx, span = GCSFuseTracer.Start(ctx, traceName, traceOpts...)
	return ctx, span
}

func (o *otelTracer) EndTrace(span trace.Span) {
	span.End()
}

func (o *otelTracer) RecordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func NewOtelTracer() TraceHandle {
	var o otelTracer
	return &o
}
