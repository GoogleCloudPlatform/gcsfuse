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
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type otelTracer struct {
	tracer trace.Tracer
}

func (o *otelTracer) StartSpan(ctx context.Context, traceName string) (context.Context, trace.Span) {
	return o.tracer.Start(ctx, traceName)
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

func (o *otelTracer) PropagateTraceContext(newCtx context.Context, oldCtx context.Context) context.Context {
	span := trace.SpanFromContext(oldCtx)
	return trace.ContextWithSpan(newCtx, span)
}

func NewOTELTracer() TraceHandle {
	return &otelTracer{
		tracer: otel.Tracer(name),
	}
}
