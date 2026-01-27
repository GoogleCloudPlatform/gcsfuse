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

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type noopTracer struct{}

func (*noopTracer) StartSpan(ctx context.Context, traceName string) (context.Context, trace.Span) {
	return ctx, noop.Span{}
}

func (*noopTracer) StartSpanLink(ctx context.Context, traceName string) (context.Context, trace.Span) {
	return ctx, noop.Span{}
}

func (*noopTracer) StartServerSpan(ctx context.Context, traceName string) (context.Context, trace.Span) {
	return ctx, noop.Span{}
}

func (*noopTracer) EndSpan(span trace.Span) {}

func (*noopTracer) RecordError(span trace.Span, err error) {}

// Return the new context as it is as this is a no-op implementation
func (*noopTracer) PropagateTraceContext(newCtx context.Context, _ context.Context) context.Context {
	return newCtx
}

func (o *noopTracer) SetCacheReadAttributes(span trace.Span, isCacheHit bool, bytesRead int) {}

func NewNoopTracer() TraceHandle {
	return new(noopTracer)
}
