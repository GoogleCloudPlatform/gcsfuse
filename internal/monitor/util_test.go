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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type MonitorUtilTest struct {
	suite.Suite
}

func TestMonitorUtilSuite(t *testing.T) {
	// The tracer provider needs to be set for the test suite.
	otel.SetTracerProvider(noop.NewTracerProvider())
	suite.Run(t, new(MonitorUtilTest))
}

func (s *MonitorUtilTest) TestAugmentTraceContext() {
	tracer := otel.Tracer("test-tracer")
	parentCtx, parentSpan := tracer.Start(context.Background(), "parent")
	defer parentSpan.End()

	testCases := []struct {
		name                  string
		tracingEnabled        bool
		expectSpanPropagation bool
	}{
		{
			name:                  "TracingEnabled_ShouldPropagateSpan",
			tracingEnabled:        true,
			expectSpanPropagation: true,
		},
		{
			name:                  "TracingDisabled_ShouldNotPropagateSpan",
			tracingEnabled:        false,
			expectSpanPropagation: false,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// Arrange
			config := &cfg.Config{}
			if tc.tracingEnabled {
				config.Monitoring.ExperimentalTracingMode = "stdout"
			}
			newCtx := context.Background()

			// Act
			augmentedCtx := AugmentTraceContext(newCtx, parentCtx, config)

			// Assert
			spanFromAugmentedCtx := trace.SpanFromContext(augmentedCtx)
			if tc.expectSpanPropagation {
				assert.Equal(t, parentSpan, spanFromAugmentedCtx, "Span should be propagated to the new context.")
			} else {
				assert.NotEqual(t, parentSpan, spanFromAugmentedCtx, "Span should not be propagated to the new context.")
				assert.False(t, spanFromAugmentedCtx.SpanContext().IsValid(), "New context should not have a valid span.")
			}
		})
	}
}
