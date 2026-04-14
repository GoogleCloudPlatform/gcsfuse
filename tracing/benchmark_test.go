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

func BenchmarkTrace(b *testing.B) {
	traceHandlers := []struct {
		prefix      string
		traceHandle TraceHandle
	}{
		{
			prefix:      "Otel",
			traceHandle: NewOTELTracer(),
		},
		{
			prefix:      "Noop",
			traceHandle: NewNoopTracer(),
		},
	}
	for _, tc := range traceHandlers {
		th := tc.traceHandle
		prefix := tc.prefix

		b.Run(fmt.Sprintf("BenchmarkStartSpan_%s", prefix), func(b *testing.B) {
			ctx := context.Background()

			for b.Loop() {
				_, span := th.StartSpan(ctx, "TestSpanName")
				th.EndSpan(span)
			}
		})

		b.Run(fmt.Sprintf("BenchmarkStartServerSpan_%s", prefix), func(b *testing.B) {
			ctx := context.Background()

			for b.Loop() {
				_, span := th.StartServerSpan(ctx, "TestSpanName")
				th.EndSpan(span)
			}
		})

		b.Run(fmt.Sprintf("BenchmarkRecordError_%s", prefix), func(b *testing.B) {
			ctx := context.Background()
			err := fmt.Errorf("TestError")
			_, span := th.StartSpan(ctx, "TestSpanName")

			for b.Loop() {
				th.RecordError(span, err)
			}

			th.EndSpan(span)
		})

		b.Run(fmt.Sprintf("BenchmarkTraceUploadWithErrorNoBytes_%s", prefix), func(b *testing.B) {
			ctx := context.Background()
			err := fmt.Errorf("TestError")
			bytes := int64(0)

			for b.Loop() {
				_, finishSpan := th.TraceUpload(ctx, "TestSpanName", "A/B/C/test_file.text", &bytes, &err)
				finishSpan()
			}
		})

		b.Run(fmt.Sprintf("BenchmarkTraceUploadWithErrorWithBytes_%s", prefix), func(b *testing.B) {
			ctx := context.Background()
			err := fmt.Errorf("TestError")
			bytes := int64(33554432)

			for b.Loop() {
				_, finishSpan := th.TraceUpload(ctx, "TestSpanName", "A/B/C/test_file.text", &bytes, &err)
				finishSpan()
			}
		})

		b.Run(fmt.Sprintf("BenchmarkTraceUploadWithoutErrorNoBytes_%s", prefix), func(b *testing.B) {
			ctx := context.Background()
			bytes := int64(0)

			for b.Loop() {
				_, finishSpan := th.TraceUpload(ctx, "TestSpanName", "A/B/C/test_file.text", &bytes, nil)
				finishSpan()
			}
		})

		b.Run(fmt.Sprintf("BenchmarkTraceUploadWithoutErrorWithBytes_%s", prefix), func(b *testing.B) {
			ctx := context.Background()
			bytes := int64(33554432)

			for b.Loop() {
				_, finishSpan := th.TraceUpload(ctx, "TestSpanName", "A/B/C/test_file.text", &bytes, nil)
				finishSpan()
			}
		})

		b.Run(fmt.Sprintf("BenchmarkSetCacheReadAttributes_%s", prefix), func(b *testing.B) {
			ctx := context.Background()

			_, span := th.StartSpan(ctx, "TestSpanName")
			for b.Loop() {
				th.SetCacheReadAttributes(span, true, 100)
			}
			th.EndSpan(span)
		})

		b.Run(fmt.Sprintf("BenchmarkSetUploadAttributes_%s", prefix), func(b *testing.B) {
			ctx := context.Background()

			_, span := th.StartSpan(ctx, "TestSpanName")
			for b.Loop() {
				th.SetUploadAttributes(span, 100, "A/B/C/test_file.text")
			}
			th.EndSpan(span)
		})

		b.Run(fmt.Sprintf("BenchmarkPropagateTraceContext_%s", prefix), func(b *testing.B) {
			ctx := context.Background()

			for b.Loop() {
				_ = th.PropagateTraceContext(ctx, ctx)
			}
		})
	}
}
