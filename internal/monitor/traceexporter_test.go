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

package monitor

import (
	"context"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/v4/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
)

func strPtr(s string) *string { return &s }

func TestSetupTracing_GoStorageDevOtelTracingEnv(t *testing.T) {
	tests := []struct {
		name           string
		traceConfig    cfg.TraceConfig
		initialEnvVal  *string
		expectedExists bool
		expectedVal    string
	}{
		{
			name: "TracingEnabled_SetsDevOtelEnv",
			traceConfig: cfg.TraceConfig{
				Exporters:     []string{"stdout"},
				SamplingRatio: 0.1,
			},
			initialEnvVal:  nil,
			expectedExists: true,
			expectedVal:    "true",
		},
		{
			name: "TracingDisabled_LeavesDevOtelEnvUnset",
			traceConfig: cfg.TraceConfig{
				Exporters:     []string{"stdout"},
				SamplingRatio: 0,
			},
			initialEnvVal:  nil,
			expectedExists: false,
		},
		{
			name: "TracingEnabled_PreservesExplicitFalseOverride",
			traceConfig: cfg.TraceConfig{
				Exporters:     []string{"stdout"},
				SamplingRatio: 0.1,
			},
			initialEnvVal:  strPtr("false"),
			expectedExists: true,
			expectedVal:    "false",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() { otel.SetTracerProvider(noop.NewTracerProvider()) })

			prevVal, hadPrev := os.LookupEnv(goStorageDevOtelTracingEnv)
			t.Cleanup(func() {
				if hadPrev {
					_ = os.Setenv(goStorageDevOtelTracingEnv, prevVal)
				} else {
					_ = os.Unsetenv(goStorageDevOtelTracingEnv)
				}
			})

			if tc.initialEnvVal == nil {
				require.NoError(t, os.Unsetenv(goStorageDevOtelTracingEnv))
			} else {
				require.NoError(t, os.Setenv(goStorageDevOtelTracingEnv, *tc.initialEnvVal))
			}

			c := &cfg.Config{Trace: tc.traceConfig}
			shutdown := SetupTracing(context.Background(), c, "test-mount-id")
			require.NotNil(t, shutdown)
			t.Cleanup(func() { _ = shutdown(context.Background()) })

			gotVal, gotExists := os.LookupEnv(goStorageDevOtelTracingEnv)
			assert.Equal(t, tc.expectedExists, gotExists)
			if tc.expectedExists {
				assert.Equal(t, tc.expectedVal, gotVal)
			}
		})
	}
}
