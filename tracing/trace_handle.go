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

// TraceHandle provides an interface for recording traces, trace links and everything related to tracing to ensure a corresponding no-op implementation
type TraceHandle interface {
	// Start a span with a given name & context
	StartSpan(ctx context.Context, traceName string) (context.Context, trace.Span)

	// Start a span of span kind server given name & context
	StartServerSpan(ctx context.Context, traceName string) (context.Context, trace.Span)

	// End a span
	EndSpan(span trace.Span)

	// Record an error on the span for export in case of failure
	RecordError(span trace.Span, err error)
}
