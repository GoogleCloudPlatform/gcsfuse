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

var NoOpTraceHandle = NewNoopTracer()

func BenchmarkNoOpTracerStartEndSpan(b *testing.B) {
	ctx := context.Background()

	for b.Loop() {
		_, span := NoOpTraceHandle.StartSpan(ctx, "TestSpanName")
		NoOpTraceHandle.EndSpan(span)
	}
}

func BenchmarkNoOpTracerStartSpanLink(b *testing.B) {
	ctx := context.Background()

	for b.Loop() {
		_, span := NoOpTraceHandle.StartServerSpan(ctx, "TestSpanName")
		NoOpTraceHandle.EndSpan(span)
	}
}

func BenchmarkNoOpTracerStartRecordErrorEndSpan(b *testing.B) {
	ctx := context.Background()

	for b.Loop() {
		_, span := NoOpTraceHandle.StartSpan(ctx, "TestSpanName")
		NoOpTraceHandle.RecordError(span, nil)
		NoOpTraceHandle.EndSpan(span)
	}
}

func BenchmarkNoOpTracerPropagateTraceContext(b *testing.B) {
	ctx := context.Background()

	for b.Loop() {
		NoOpTraceHandle.PropagateTraceContext(ctx, ctx)
	}
}
