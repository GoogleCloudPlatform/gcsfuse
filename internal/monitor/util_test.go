// Copyright 2025 Google LLC
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
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type MonitorUtilTest struct {
	suite.Suite
}

func TestMonitorUtilSuite(t *testing.T) {
	suite.Run(t, new(MonitorUtilTest))
}

func (s *MonitorUtilTest) TestGetTraceContext() {
	// Setup a test tracer provider for OpenTelemetry. This allows us to create
	// and manage test spans.
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := otel.Tracer("test-tracer")

	testCases := []struct {
		name                string
		isTracingEnabled    bool
		inputContextHasSpan bool
	}{
		{
			name:                "TracingEnabled_WithSpanInInputContext",
			isTracingEnabled:    true,
			inputContextHasSpan: true,
		},
		{
			name:                "TracingEnabled_NoSpanInInputContext",
			isTracingEnabled:    true,
			inputContextHasSpan: false,
		},
		{
			name:                "TracingDisabled_WithSpanInInputContext",
			isTracingEnabled:    false,
			inputContextHasSpan: true,
		},
		{
			name:                "TracingDisabled_NoSpanInInputContext",
			isTracingEnabled:    false,
			inputContextHasSpan: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Arrange
			var inputCtx context.Context
			var originalSpan oteltrace.Span
			var originalSpanContext oteltrace.SpanContext

			if tc.inputContextHasSpan {
				// Create an input context with an active span.
				inputCtx, originalSpan = tracer.Start(context.Background(), "original-span")
				originalSpanContext = originalSpan.SpanContext()
			} else {
				// Create a plain background context without a span.
				inputCtx = context.Background()
				originalSpanContext = oteltrace.SpanContext{} // Represents an invalid/no-op span context.
			}

			// Configure the mock `cfg.Config` based on the test case.
			testConfig := &cfg.Config{
				Monitoring: cfg.MonitoringConfig{
					ExperimentalTracingMode: "", // Default to disabled.
				},
			}
			if tc.isTracingEnabled {
				testConfig.Monitoring.ExperimentalTracingMode = "stdout" // Any non-empty value enables tracing.
			}

			// Act
			resultCtx := GetTraceContext(inputCtx, testConfig)

			// Assert
			resultSpan := oteltrace.SpanFromContext(resultCtx)
			resultSpanContext := resultSpan.SpanContext()

			if tc.isTracingEnabled && tc.inputContextHasSpan {
				s.Assert().True(resultSpanContext.IsValid(), "Result span should be valid when tracing is enabled and input had a span")
				s.Assert().Equal(originalSpanContext.TraceID(), resultSpanContext.TraceID(), "TraceID should match original when tracing is enabled and input had a span")
				s.Assert().Equal(originalSpanContext.SpanID(), resultSpanContext.SpanID(), "SpanID should match original when tracing is enabled and input had a span")
			} else {
				// In all other cases (tracing disabled, or tracing enabled but no span in input),
				// the resulting context should either not have a valid span or have a no-op span.
				s.Assert().False(resultSpanContext.IsValid(), "Result span should be invalid when tracing is disabled or no span in input context")
				// If tracing is disabled but input had a span, ensure a *new* background context is returned, not the original.
				if !tc.isTracingEnabled && tc.inputContextHasSpan {
					s.Assert().NotSame(inputCtx, resultCtx, "Contexts should be different when tracing is disabled but input had span")
				}
			}
		})
	}
}
