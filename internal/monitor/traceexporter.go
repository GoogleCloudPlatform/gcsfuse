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

	"fmt"

	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// SetupTracing bootstraps the OpenTelemetry tracing pipeline.
func SetupTracing(ctx context.Context, c *cfg.Config, mountID string) common.ShutdownFn {
	tp, shutdown, err := newTraceProvider(ctx, c, mountID)

	otel.SetTextMapPropagator(propagation.TraceContext{})

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

func newTraceProvider(ctx context.Context, c *cfg.Config, mountID string) (trace.TracerProvider, common.ShutdownFn, error) {
	switch c.Monitoring.ExperimentalTracingMode {
	case "stdout":
		return newStdoutTraceProvider()
	case "gcptrace":
		return newGCPCloudTraceExporter(ctx, c, mountID)
	case "gcptrace-debug":
		return newGCPCloudTraceDebugExporter(ctx, c, mountID)
	default:
		return nil, nil, nil
	}
}

func newStdoutTraceProvider() (trace.TracerProvider, common.ShutdownFn, error) {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
	return tp, tp.Shutdown, nil
}

func newGCPCloudTraceExporter(ctx context.Context, c *cfg.Config, mountID string) (*sdktrace.TracerProvider, common.ShutdownFn, error) {
	var traceOptions []cloudtrace.Option

	if c.Monitoring.ExperimentalTracingProjectId != "" {
		traceOptions = append(traceOptions, cloudtrace.WithProjectID(c.Monitoring.ExperimentalTracingProjectId))
	}

	exporter, err := cloudtrace.New(traceOptions...)

	if err != nil {
		return nil, nil, err
	}
	res, err := getResource(ctx, mountID)
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res), sdktrace.WithSampler(sdktrace.TraceIDRatioBased(c.Monitoring.ExperimentalTracingSamplingRatio)))

	return tp, tp.Shutdown, nil
}

func newGCPCloudTraceDebugExporter(ctx context.Context, c *cfg.Config, mountID string) (*sdktrace.TracerProvider, common.ShutdownFn, error) {
	var traceOptions []cloudtrace.Option

	if c.Monitoring.ExperimentalTracingProjectId != "" {
		traceOptions = append(traceOptions, cloudtrace.WithProjectID(c.Monitoring.ExperimentalTracingProjectId))
	}

	exporter, err := cloudtrace.New(traceOptions...)

	if err != nil {
		return nil, nil, err
	}
	res, err := getResource(ctx, mountID)
	if err != nil {
		return nil, nil, err
	}

	globalDebugger := NewOrphanDebugger()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(globalDebugger),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exporter)),
		sdktrace.WithResource(res), sdktrace.WithSampler(sdktrace.TraceIDRatioBased(c.Monitoring.ExperimentalTracingSamplingRatio)))

	var cleanupFunc common.ShutdownFn

	cleanupFunc = func(ctx context.Context) error {

		logger.Info("Starting graceful OpenTelemetry shutdown...")

		// Step 1: Force Flush
		// This processes all spans that *did* call span.End() and updates the debugger's map.
		if err := tp.ForceFlush(ctx); err != nil {
			logger.Errorf("Error flushing spans before audit: %v\n", err)
			return err
		}

		// Step 2: Run the Orphan Audit (FindOrphans)
		if globalDebugger != nil {
			const orphanReportFile = "final_orphan_spans_report.json"
			if err := globalDebugger.FindOrphans(orphanReportFile); err != nil {
				// Log the error but continue to the final shutdown
				logger.Errorf("Error generating orphan report: %v\n", err)
				return err
			}
		}

		// Step 3: Call the TracerProvider's Shutdown
		// This closes connections and stops background workers.
		if err := tp.Shutdown(ctx); err != nil {
			fmt.Printf("Error shutting down TracerProvider: %v\n", err)
			return err
		}

		fmt.Println("OpenTelemetry shutdown complete.")

		return nil
	}

	return tp, cleanupFunc, nil
}
