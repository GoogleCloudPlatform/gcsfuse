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
	"testing"
)

// Otel Trace Handle benchmark tests

var OtelTraceHandle = NewOTELTracer()

func BenchmarkOtelTracerSetCacheReadAttributes(b *testing.B) {
	ctx := context.Background()
	_, span := OtelTraceHandle.StartSpan(ctx, "TestSpanName")

	for b.Loop() {
		OtelTraceHandle.SetCacheReadAttributes(span, true, 100)
	}
	OtelTraceHandle.EndSpan(span)
}

// Noop Trace Handle benchmark tests

var NoopTraceHandle = NewNoopTracer()

func BenchmarkNoopTracerStartSpan(b *testing.B) {
	ctx := context.Background()
	for b.Loop() {
		_, span := NoopTraceHandle.StartSpan(ctx, "TestSpanName")
		NoopTraceHandle.EndSpan(span)
	}
}

func BenchmarkNoopTracerStartServerSpan(b *testing.B) {
	ctx := context.Background()
	for b.Loop() {
		_, span := NoopTraceHandle.StartServerSpan(ctx, "TestSpanName")
		NoopTraceHandle.EndSpan(span)
	}
}

func BenchmarkNoopTracerRecordError(b *testing.B) {
	ctx := context.Background()
	_, span := NoopTraceHandle.StartSpan(ctx, "TestSpanName")
	for b.Loop() {
		NoopTraceHandle.RecordError(span, nil)
	}
	NoopTraceHandle.EndSpan(span)
}

func BenchmarkNoopTracerSetCacheReadAttributes(b *testing.B) {
	ctx := context.Background()
	_, span := NoopTraceHandle.StartSpan(ctx, "TestSpanName")

	for b.Loop() {
		NoopTraceHandle.SetCacheReadAttributes(span, true, 100)
	}
	NoopTraceHandle.EndSpan(span)
}

func BenchmarkNoopTracerPropagateTraceContext(b *testing.B) {
	ctx := context.Background()
	for b.Loop() {
		_ = NoopTraceHandle.PropagateTraceContext(ctx, ctx)
	}
}
