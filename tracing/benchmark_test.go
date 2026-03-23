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
	"fmt"
	"testing"
)

var traceHandlers = map[string]TraceHandle{
	"otel": NewOTELTracer(),
	"noop": NewNoopTracer(),
}

func runTraceHandleBenchmarks(b *testing.B, benchFn func(b *testing.B, th TraceHandle)) {
	for name, th := range traceHandlers {
		b.Run(name, func(b *testing.B) {
			benchFn(b, th)
		})
	}
}

func BenchmarkStartSpan(b *testing.B) {
	ctx := context.Background()
	runTraceHandleBenchmarks(b, func(b *testing.B, th TraceHandle) {
		for b.Loop() {
			_, span := th.StartSpan(ctx, "TestSpanName")
			th.EndSpan(span)
		}
	})
}

func BenchmarkStartServerSpan(b *testing.B) {
	ctx := context.Background()
	runTraceHandleBenchmarks(b, func(b *testing.B, th TraceHandle) {
		for b.Loop() {
			_, span := th.StartServerSpan(ctx, "TestSpanName")
			th.EndSpan(span)
		}
	})
}

func BenchmarkRecordError(b *testing.B) {
	ctx := context.Background()
	err := fmt.Errorf("test error")
	runTraceHandleBenchmarks(b, func(b *testing.B, th TraceHandle) {
		_, span := th.StartSpan(ctx, "TestSpanName")
		for b.Loop() {
			th.RecordError(span, err)
		}
		th.EndSpan(span)
	})
}

func BenchmarkSetCacheReadAttributes(b *testing.B) {
	ctx := context.Background()
	runTraceHandleBenchmarks(b, func(b *testing.B, th TraceHandle) {
		_, span := th.StartSpan(ctx, "TestSpanName")
		for b.Loop() {
			th.SetCacheReadAttributes(span, true, 100)
		}
		th.EndSpan(span)
	})
}

func BenchmarkPropagateTraceContext(b *testing.B) {
	ctx := context.Background()
	runTraceHandleBenchmarks(b, func(b *testing.B, th TraceHandle) {
		for b.Loop() {
			_ = th.PropagateTraceContext(ctx, ctx)
		}
	})
}
