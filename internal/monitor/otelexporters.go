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
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	cloudmetric "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const serviceName = "gcsfuse"
const cloudMonitoringMetricPrefix = "custom.googleapis.com/gcsfuse/"

var allowedMetricPrefixes = []string{"fs/", "gcs/", "file_cache/", "buffered_read/", "grpc."}

// SetupOTelMetricExporters sets up the metrics exporters
func SetupOTelMetricExporters(ctx context.Context, c *cfg.Config, mountID string) (shutdownFn common.ShutdownFn) {
	var shutdownFns []common.ShutdownFn
	options := make([]metric.Option, 0)

	opts, promShutdownFn := setupPrometheus(c.Metrics.PrometheusPort)
	options = append(options, opts...)
	shutdownFns = append(shutdownFns, promShutdownFn)

	opts = setupCloudMonitoring(c.Metrics.CloudMetricsExportIntervalSecs)
	options = append(options, opts...)

	res, err := getResource(ctx, mountID)
	if err != nil {
		logger.Errorf("Error while fetching resource: %v", err)
	} else {
		options = append(options, metric.WithResource(res))
	}

	options = append(options, metric.WithView(dropDisallowedMetricsView), metric.WithExemplarFilter(exemplar.AlwaysOffFilter))

	meterProvider := metric.NewMeterProvider(options...)

	otel.SetMeterProvider(meterProvider)

	shutdownFns = append(shutdownFns, meterProvider.Shutdown)

	return common.JoinShutdownFunc(shutdownFns...)
}

// dropUnwantedMetricsView is an OTel View that drops the metrics that don't match the allowed prefixes.
func dropDisallowedMetricsView(i metric.Instrument) (metric.Stream, bool) {
	s := metric.Stream{Name: i.Name, Description: i.Description, Unit: i.Unit}
	for _, prefix := range allowedMetricPrefixes {
		if strings.HasPrefix(i.Name, prefix) {
			return s, true
		}
	}
	s.Aggregation = metric.AggregationDrop{}
	return s, true
}

func setupCloudMonitoring(secs int64) []metric.Option {
	if secs <= 0 {
		return nil
	}
	options := []cloudmetric.Option{
		cloudmetric.WithMetricDescriptorTypeFormatter(metricFormatter),
	}
	exporter, err := cloudmetric.New(options...)
	if err != nil {
		logger.Errorf("Error while creating Google Cloud exporter:%v", err)
		return nil
	}

	// Wrap the exporter to handle permission denied errors
	wrappedExporter := &permissionAwareExporter{
		Exporter: exporter,
	}

	reader := metric.NewPeriodicReader(wrappedExporter, metric.WithInterval(time.Duration(secs)*time.Second))
	return []metric.Option{metric.WithReader(reader)}
}

// permissionAwareExporter wraps a metric.Exporter and disables itself if it encounters
// a PermissionDenied error. This prevents log spam when the environment lacks
// necessary permissions.
type permissionAwareExporter struct {
	metric.Exporter
	// disabled indicates whether the exporter has been permanently disabled.
	disabled atomic.Bool
}

func (e *permissionAwareExporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	// Check if disabled before attempting export to save resources and avoid noise.
	if e.disabled.Load() {
		return nil
	}

	err := e.Exporter.Export(ctx, rm)
	// If we get a PermissionDenied error, disable the exporter to prevent future attempts.
	if err != nil && status.Code(err) == codes.PermissionDenied {
		if e.disabled.CompareAndSwap(false, true) {
			logger.Errorf("Disabling Cloud Monitoring exporter due to permission denied error: %v", err)
		}
	}
	return err
}

func (e *permissionAwareExporter) ForceFlush(ctx context.Context) error {
	if e.disabled.Load() {
		return nil
	}
	return e.Exporter.ForceFlush(ctx)
}

func metricFormatter(m metricdata.Metrics) string {
	return cloudMonitoringMetricPrefix + strings.ReplaceAll(m.Name, ".", "/")
}
func setupPrometheus(port int64) ([]metric.Option, common.ShutdownFn) {
	if port <= 0 {
		return nil, nil
	}
	exporter, err := prometheus.New(prometheus.WithoutUnits(), prometheus.WithoutCounterSuffixes(), prometheus.WithoutScopeInfo(), prometheus.WithoutTargetInfo())
	if err != nil {
		logger.Errorf("Error while creating prometheus exporter:%v", err)
		return nil, nil
	}
	shutdownCh := make(chan context.Context)
	done := make(chan any)
	go serveMetrics(port, shutdownCh, done)
	return []metric.Option{metric.WithReader(exporter)}, func(ctx context.Context) error {
		shutdownCh <- ctx
		close(shutdownCh)
		<-done
		close(done)
		return nil
	}
}

func serveMetrics(port int64, shutdownCh <-chan context.Context, done chan<- any) {
	logger.Infof("Serving metrics at localhost:%d/metrics", port)
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
		ctx := <-shutdownCh
		defer func() { done <- true }()
		logger.Info("Shutting down Prometheus exporter.")
		if err := prometheusServer.Shutdown(ctx); err != nil {
			logger.Errorf("Error while shutting down Prometheus exporter:%v", err)
			return
		}
		logger.Info("Prometheus exporter shutdown")
	}()
	logger.Info("Prometheus collector exporter started")
}

func getResource(ctx context.Context, mountID string) (*resource.Resource, error) {
	return resource.New(ctx,
		// Use the GCP resource detector to detect information about the GCP platform
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(common.GetVersion()),
			semconv.ServiceInstanceID(mountID),
		),
	)
}
