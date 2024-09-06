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

	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/detectors/gcp"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type ShutdownFn func(ctx context.Context) error

// SetupTracing bootstraps the OpenTelemetry tracing pipeline.
func SetupTracing(ctx context.Context, c *cfg.Config) ShutdownFn {
	tp, shutdown, err := newTraceProvider(ctx, c)
	if err == nil && tp != nil {
		otel.SetTracerProvider(tp)
		return shutdown
	}
	return nil
}

func newTraceProvider(ctx context.Context, c *cfg.Config) (trace.TracerProvider, ShutdownFn, error) {
	switch c.Monitoring.ExperimentalTracingMode {
	case "stdout":
		return newStdoutTraceProvider()
	case "gcptrace":
		return newGCPCloudTraceExporter(ctx, c)
	default:
		return nil, nil, nil
	}
}
func newStdoutTraceProvider() (trace.TracerProvider, ShutdownFn, error) {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
	return tp, tp.Shutdown, nil
}

func newGCPCloudTraceExporter(ctx context.Context, c *cfg.Config) (*sdktrace.TracerProvider, ShutdownFn, error) {
	exporter, err := cloudtrace.New()
	if err != nil {
		return nil, nil, err
	}
	appName := "gcsfuse"
	if c.AppName != "" {
		appName = c.AppName
	}
	res, err := resource.New(ctx,
		// Use the GCP resource detector to detect information about the GCP platform
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName(appName),
		),
	)
	if err != nil {
		return nil, nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res), sdktrace.WithSampler(sdktrace.TraceIDRatioBased(c.Monitoring.ExperimentalTracingSamplingRatio)))

	return tp, tp.Shutdown, nil
}
