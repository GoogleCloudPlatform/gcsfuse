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
	"log"
	"net/http"
	"time"

	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"

	cloudmetric "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/detectors/gcp"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func SetupOTelSDK(ctx context.Context, c *cfg.Config) (shutdown common.ShutdownFn) {
	if shutdownFn := setupTracing(ctx, c); shutdownFn != nil {
		shutdown = common.JoinShutdownFunc(shutdown, shutdownFn)
	}

	if shutdownFn := setupMetrics(ctx, c); shutdownFn != nil {
		shutdown = common.JoinShutdownFunc(shutdown, shutdownFn)
	}
	return shutdown
}
func setupCloudMetricsExporter(c *cfg.Config) ([]metric.Option, error) {
	opts := make([]metric.Option, 0)
	exporter, err := cloudmetric.New()
	if err != nil {
		return nil, err
	}
	opts = append(opts, metric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(time.Duration(c.Metrics.ExportIntervalSecs)*time.Second))))
	return opts, nil
}
func setupPrometheus(port int64) ([]metric.Option, common.ShutdownFn) {
	exporter, err := prometheus.New(prometheus.WithoutUnits(), prometheus.WithoutCounterSuffixes(), prometheus.WithoutScopeInfo(), prometheus.WithoutTargetInfo())
	if err != nil {
		logger.Errorf("Error while creating prometheus exporter")
		return nil, nil
	}

	ch := make(chan context.Context)
	go serveMetrics(port, ch)

	return []metric.Option{metric.WithReader(exporter)}, func(ctx context.Context) error {
		ch <- ctx
		return nil
	}
}
func setupMetrics(ctx context.Context, c *cfg.Config) common.ShutdownFn {
	options := make([]metric.Option, 0)
	if promPort := c.Metrics.PrometheusPort; promPort > 0 {
		opts, err := setupPrometheus(c.Metrics.PrometheusPort)
		if err != nil {
			logger.Errorf("Error while starting up Prometheus exporter: %v", err)
		} else {
			options = append(options, opts...)
		}
	}

	if exportInterval := c.Metrics.ExportIntervalSecs; exportInterval > 0 {
		opts, err := setupCloudMetricsExporter(c)
		if err != nil {
			logger.Errorf("Error while starting up Cloud exporter: %v", err)
		} else {
			options = append(options, opts...)
		}
	}
	res, err := getResource(ctx, c)
	if err != nil {
		logger.Errorf("Error while fetching resource: %v", err)
	} else {
		options = append(options, metric.WithResource(res))
	}
	meterProvider := metric.NewMeterProvider(options...)
	otel.SetMeterProvider(meterProvider)
	return meterProvider.Shutdown
}

func serveMetrics(port int64, done <-chan context.Context) {
	log.Printf("serving metrics at localhost:%d/metrics", port)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	prometheusServer := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := prometheusServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Failed to start Prometheus server: %v", err)
		}
	}()

	go func() {
		ctx := <-done
		if err := prometheusServer.Shutdown(ctx); err != nil {
			logger.Errorf("Error while shutting down Prometheus exporter:%v", err)
			return
		}
		logger.Info("Prometheus exporter shutdown")
	}()

	logger.Info("Prometheus collector exporter started")
}

func setupTracing(ctx context.Context, c *cfg.Config) common.ShutdownFn {
	tp, shutdown, err := newTraceProvider(ctx, c)
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

func newTraceProvider(ctx context.Context, c *cfg.Config) (trace.TracerProvider, common.ShutdownFn, error) {
	switch c.Monitoring.ExperimentalTracingMode {
	case "stdout":
		return newStdoutTraceProvider()
	case "gcptrace":
		return newGCPCloudTraceExporter(ctx, c)
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

func getAppName(c *cfg.Config) string {
	appName := "gcsfuse"
	if c.AppName != "" {
		appName = c.AppName
	}
	return appName
}

func getResource(ctx context.Context, c *cfg.Config) (*resource.Resource, error) {
	return resource.New(ctx,
		// Use the GCP resource detector to detect information about the GCP platform
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName(getAppName(c)),
			semconv.ServiceVersion(common.GetVersion()),
		),
	)
}
func newGCPCloudTraceExporter(ctx context.Context, c *cfg.Config) (*sdktrace.TracerProvider, common.ShutdownFn, error) {
	exporter, err := cloudtrace.New()
	if err != nil {
		return nil, nil, err
	}

	res, err := getResource(ctx, c)
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res), sdktrace.WithSampler(sdktrace.TraceIDRatioBased(c.Monitoring.ExperimentalTracingSamplingRatio)))

	return tp, tp.Shutdown, nil
}
