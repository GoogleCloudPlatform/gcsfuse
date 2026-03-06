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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Identifies the span link as a parent relationship for proper trace visualization in GCP.
var parentLinkAttributes = []attribute.KeyValue{
	attribute.Int64("gcp.cloud_trace.link_type", 1),
}

type otelTracer struct {
	tracer trace.Tracer
}

func (o *otelTracer) StartSpan(ctx context.Context, traceName string) (context.Context, trace.Span) {
	return o.tracer.Start(ctx, traceName)
}

func (o *otelTracer) StartSpanLink(ctx context.Context, traceName string) (context.Context, trace.Span) {
	span := trace.SpanFromContext(ctx)
	traceOpts := []trace.SpanStartOption{
		trace.WithLinks(trace.Link{
			SpanContext: span.SpanContext(),
			Attributes:  parentLinkAttributes,
		}),
	}
	return o.tracer.Start(ctx, traceName, traceOpts...)
}

func (o *otelTracer) StartServerSpan(ctx context.Context, traceName string) (context.Context, trace.Span) {
	return o.tracer.Start(ctx, traceName, trace.WithSpanKind(trace.SpanKindServer))
}

func (o *otelTracer) EndSpan(span trace.Span) {
	span.End()
}

func (o *otelTracer) RecordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// This method is performant in the no-op implementation as the attribute.Bool and int methods perform heap allocation and they go as bad as 60-90 ns
// It is not good that way so the whole attribute creation has been moved inside this method keeping no-op lightweight
// Performance characteristics for this method along with StartSpan & EndSpan together for no-op
func (o *otelTracer) SetCacheReadAttributes(span trace.Span, isCacheHit bool, bytesRead int) {
	span.SetAttributes(
		attribute.Bool(IS_CACHE_HIT, isCacheHit),
		attribute.Int(BYTES_READ, bytesRead),
	)
}

func (o *otelTracer) PropagateTraceContext(newCtx context.Context, oldCtx context.Context) context.Context {
	span := trace.SpanFromContext(oldCtx)
	return trace.ContextWithSpan(newCtx, span)
}

func NewOTELTracer() TraceHandle {
	return &otelTracer{
		tracer: otel.Tracer(name),
	}
}
