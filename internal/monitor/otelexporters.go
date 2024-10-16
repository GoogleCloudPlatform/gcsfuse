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
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	cloudmetric "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	cloudtrace "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/detectors/gcp"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type ShutdownFn func(ctx context.Context) error

func SetupOTelSDK(ctx context.Context, c *cfg.Config) (shutdown ShutdownFn) {
	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	if shutdownFn := setupTracing(ctx, c); shutdownFn != nil {
		shutdownFuncs = append(shutdownFuncs, shutdownFn)
	}

	if shutdownFn := setupMetrics(ctx, c); shutdownFn != nil {
		shutdownFuncs = append(shutdownFuncs, shutdownFn)
	}
	return shutdown
}

func setupMetrics(ctx context.Context, c *cfg.Config) ShutdownFn {
	if c.Metrics.PrometheusPort != 0 {
		exporter, err := prometheus.New(prometheus.WithoutUnits(), prometheus.WithoutCounterSuffixes(), prometheus.WithoutScopeInfo(), prometheus.WithoutTargetInfo())
		if err != nil {
			logger.Errorf("Error while creating prometheus exporter")
			return nil
		}

		meterProvider := metric.NewMeterProvider(
			metric.WithReader(exporter),
		)
		otel.SetMeterProvider(meterProvider)
		ch := make(chan context.Context)
		go serveMetrics(c.Metrics.PrometheusPort, ch)
		return func(ctx context.Context) error {
			ch <- ctx
			return nil
		}
	}
	if c.Metrics.StackdriverExportInterval > 0 {
		sdkmetric.WithInterval(c.Metrics.StackdriverExportInterval)
		options := []cloudmetric.Option{
			cloudmetric.WithMetricDescriptorTypeFormatter(metricFormatter),
			//cloudmetric.WithMonitoredResourceDescription(),
			//cloudmetric.WithCreateServiceTimeSeries(),
		}
		exporter, err := cloudmetric.New(options...)
		if err != nil {
			logger.Errorf("Error while creating Google Cloud exporter:%v", err)
			return nil
		}
		appName := "gcsfuse"
		if c.AppName != "" {
			appName = c.AppName
		}
		res, err := resourceObj(ctx, appName)
		if err != nil {
			logger.Errorf("error while creating resource object:%v", res)
		}
		r := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(c.Metrics.StackdriverExportInterval))
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(r),
			sdkmetric.WithResource(res))
		otel.SetMeterProvider(mp)
		logger.Info("gcp metrics exporter started")
		return func(ctx context.Context) error { return mp.Shutdown(ctx) }
	}
	return nil
}

func resourceObj(ctx context.Context, appName string) (*resource.Resource, error) {
	return resource.New(ctx,
		// Use the GCP resource detector to detect information about the GCP platform
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
		resource.WithProcess(),
		resource.WithAttributes(
			semconv.ServiceName(appName),
			semconv.ServiceVersion(common.GetVersion()),
		),
	)
}

func metricFormatter(m metricdata.Metrics) string {
	return "custom.googleapis.com/gcsfuse/" + strings.ReplaceAll(m.Name, ".", "/")
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

func setupTracing(ctx context.Context, c *cfg.Config) ShutdownFn {
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
	res, err := resourceObj(ctx, appName)
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res), sdktrace.WithSampler(sdktrace.TraceIDRatioBased(c.Monitoring.ExperimentalTracingSamplingRatio)))
	return tp, tp.Shutdown, nil
}
