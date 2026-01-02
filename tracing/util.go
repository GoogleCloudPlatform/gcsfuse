// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

// When tracing is enabled ensure span & trace context from oldCtx is passed on to newCtx
func MaybePropagateTraceContext(newCtx context.Context, oldCtx context.Context, isTracingEnabled bool) context.Context {
	if !isTracingEnabled {
		return newCtx
	}

	span := trace.SpanFromContext(oldCtx)
	return trace.ContextWithSpan(newCtx, span)
}
