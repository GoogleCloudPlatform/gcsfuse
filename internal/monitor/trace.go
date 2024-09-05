// Copyright 2024 Google LLC
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

package monitor

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TODO: name is subject to change.
const name = "cloud.google.com/gcsfuse"

var (
	tracer = noop.NewTracerProvider().Tracer("noop")
	once   sync.Once
)

// StartSpan creates a new span and returns it along with the context that's augmented with the newly-created span..
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	once.Do(func() {
		// Expect that the tracer provider has been configured by the SDK before the first call to StartSpan is made.
		ResetTracer()
	})
	return tracer.Start(ctx, spanName, opts...)
}

func ResetTracer() {
	tracer = otel.GetTracerProvider().Tracer(name)
}
