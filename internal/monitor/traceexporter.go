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
	"strings"

	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// SetupTracing bootstraps the OpenTelemetry tracing pipeline.
func SetupTracing(ctx context.Context, c *cfg.Config, mountID string) common.ShutdownFn {
	tp, shutdown, err := newTraceProvider(ctx, c, mountID)
	if err != nil {
		logger.Errorf("error occurred while setting up tracing: %v", err)
		return nil
	}
	if tp != nil {
		otel.SetTracerProvider(tp)
		return shutdown
	}

	return nil
}

type ExporterFactory func() (sdktrace.SpanExporter, error)

func newTraceProvider(ctx context.Context, c *cfg.Config, mountID string) (trace.TracerProvider, common.ShutdownFn, error) {
	var opts []sdktrace.TracerProviderOption
	exporterNames := strings.Split(c.Monitoring.ExperimentalTracingMode, ",")

	exporterRegistry := map[string]ExporterFactory{
		"stdout": func() (sdktrace.SpanExporter, error) {
			return newStdoutTraceExporter()
		},
		"gcptrace": func() (sdktrace.SpanExporter, error) {
			return newGCPCloudTraceExporter(c)
		},
	}

	for _, name := range exporterNames {
		if exporterFactory, ok := exporterRegistry[name]; ok {
			exporter, err := exporterFactory()

			if err != nil {
				logger.Errorf("failed to init %s exporter: %w", name, err)
				return nil, nil, err
			}

			opts = append(opts, sdktrace.WithBatcher(exporter))
		}
	}

	res, err := getResource(ctx, mountID)
	if err != nil {
		return nil, nil, err
	}

	opts = append(opts, sdktrace.WithResource(res), sdktrace.WithSampler(sdktrace.TraceIDRatioBased(c.Monitoring.ExperimentalTracingSamplingRatio)))

	tp := sdktrace.NewTracerProvider(opts...)

	return tp, tp.Shutdown, nil
}

func newStdoutTraceExporter() (sdktrace.SpanExporter, error) {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint())

	if err != nil {
		return nil, err
	}

	return exporter, nil
}

func newGCPCloudTraceExporter(c *cfg.Config) (sdktrace.SpanExporter, error) {
	var traceOptions []cloudtrace.Option

	if c.Monitoring.ExperimentalTracingProjectId != "" {
		traceOptions = append(traceOptions, cloudtrace.WithProjectID(c.Monitoring.ExperimentalTracingProjectId))
	}

	exporter, err := cloudtrace.New(traceOptions...)

	if err != nil {
		return nil, err
	}

	return exporter, nil
}
