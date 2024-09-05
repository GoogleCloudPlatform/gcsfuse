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

	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// TODO: name is subject to change.
const name = "cloud.google.com/gcsfuse"

var (
	tracer trace.Tracer
	once   sync.Once
)

// StartSpan creates a new span and returns it along with the context that's augmented with the newly-created span..
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	once.Do(func() {
		// Expect that the tracer provider has been configured by the SDK before the first call to StartSpan is made.
		InitializeTracer()
	})
	return tracer.Start(ctx, spanName, opts...)
}

// InitializeTracer initializes/re-initializes the tracer with the one returned by otel.GetTracerProvider().
// Its primary utility is to re-initialize the tracer in test cases.
// In main code, the tracer is initialized once and never again but in test-code,
// StartSpan can be called before the TracerProvider is set and therefore would have been set for
// the entire test runs had there not been a way to reset it especially for tests that test tracing logic.
func InitializeTracer() {
	tracer = otel.Tracer(name, trace.WithInstrumentationVersion(common.GetVersion()))
}
