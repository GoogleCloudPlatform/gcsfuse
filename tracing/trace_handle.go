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
)

const name = "cloud.google.com/gcsfuse"

// TraceHandle provides an interface for recording traces, trace links and everything related to tracing. This allows easier switching between various trace-implementations, especially with a custom no-op tracer.
type TraceHandle interface {
	// Start a span with a given name & context
	StartSpan(ctx context.Context, traceName string) (context.Context, trace.Span)

	// Start a span of span kind server given name & context
	StartServerSpan(ctx context.Context, traceName string) (context.Context, trace.Span)

	// End a span
	EndSpan(span trace.Span)

	// Record an error on the span for export in case of failure
	RecordError(span trace.Span, err error)

	// Trace starts a span and returns a finisher function.
	// Use it like: ctx, span, end_span := th.Trace(ctx, name, &err); defer end_span()
	Trace(ctx context.Context, name string, err *error) (context.Context, trace.Span, func())

	// A handle interface method to set attributes for file cache read
	// attribute creation and generic interface using variadic operator is a costly affair both from memory allocation and CPU time perspectives - (3.883 ns/op	       0 B/op	       0 allocs/op) vs (90.21 ns/op	     128 B/op	       1 allocs/op)
	// This method is specifically created so that the caller doesn't have to create the attributes themselves.
	// Instead the implementation of the TraceHandle that's chosen decides whether to create the attributes.
	// This allows skipping the attribute creation entirely in case of noop tracer which is selected when tracing is disabled.
	SetCacheReadAttributes(span trace.Span, isCacheHit bool, bytesRead int)

	// A handle interface method to set attributes for upload
	SetUploadAttributes(span trace.Span, bytesUploaded int64, objectName string)

	// A handle interface method to retain relevant span data in new context from the older context
	PropagateTraceContext(newCtx context.Context, oldCtx context.Context) context.Context
}
