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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func TestSetupTracing_SetsTraceContextPropagator(t *testing.T) {
	// Arrange
	c := &cfg.Config{
		Monitoring: cfg.MonitoringConfig{
			ExperimentalTracingMode: "stdout",
		},
	}
	ctx := context.Background()

	// Act
	shutdown := SetupTracing(ctx, c, "test-mount-id")
	defer func() {
		if shutdown != nil {
			shutdown(ctx)
		}
	}()

	// Assert
	propagator := otel.GetTextMapPropagator()
	assert.NotNil(t, propagator)
	assert.IsType(t, propagation.TraceContext{}, propagator)

	fields := propagator.Fields()
	assert.Contains(t, fields, "traceparent")
	assert.Contains(t, fields, "tracestate")
}
