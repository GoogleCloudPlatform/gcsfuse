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
	"github.com/googlecloudplatform/gcsfuse/v3/internal/auth"
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
	"google.golang.org/api/option"
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

	opts = setupCloudMonitoring(ctx, c)
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

// setupCloudMonitoring creates and configures a Cloud Monitoring metrics exporter.
//
// The function attempts to initialize the exporter with the following credential
// sources in order:
//  1. Application Default Credentials (ADC)
//  2. Token URL credentials (if configured via --token-url)
//
// When using token URL credentials, it will optionally include the billing project
// ID if specified, and gracefully falls back to no project ID if that fails.
//
// Returns nil if the export interval is not configured or if initialization fails.
func setupCloudMonitoring(ctx context.Context, c *cfg.Config) []metric.Option {
	if c.Metrics.CloudMetricsExportIntervalSecs <= 0 {
		return nil
	}

	exporter, err := createCloudMonitoringExporter(ctx, c)
	if err != nil {
		logger.Errorf("Failed to create Cloud Monitoring exporter: %v", err)
		return nil
	}

	// Wrap the exporter to handle permission denied errors gracefully.
	wrappedExporter := &permissionAwareExporter{Exporter: exporter}

	interval := time.Duration(c.Metrics.CloudMetricsExportIntervalSecs) * time.Second
	reader := metric.NewPeriodicReader(wrappedExporter, metric.WithInterval(interval))
	return []metric.Option{metric.WithReader(reader)}
}

// createCloudMonitoringExporter attempts to create a Cloud Monitoring exporter
// with multiple credential sources.
func createCloudMonitoringExporter(ctx context.Context, c *cfg.Config) (metric.Exporter, error) {
	baseOptions := []cloudmetric.Option{
		cloudmetric.WithMetricDescriptorTypeFormatter(metricFormatter),
	}

	// Attempt to create exporter with default credentials.
	exporter, err := cloudmetric.New(baseOptions...)
	if err == nil {
		logger.Infof("Cloud Monitoring exporter initialized with default credentials")
		return exporter, nil
	}

	// Fall back to token URL credentials if configured.
	if c.GcsAuth.TokenUrl == "" {
		return nil, fmt.Errorf("default credentials unavailable and no --token-url configured: %w", err)
	}

	logger.Infof("Default credentials unavailable, using --token-url for Cloud Monitoring")
	return createCloudMonitoringExporterWithTokenURL(ctx, c, baseOptions)
}

// createCloudMonitoringExporterWithTokenURL creates a Cloud Monitoring exporter
// using token URL credentials.
func createCloudMonitoringExporterWithTokenURL(
	ctx context.Context,
	c *cfg.Config,
	baseOptions []cloudmetric.Option,
) (metric.Exporter, error) {
	tokenSrc, err := auth.NewTokenSourceFromURL(ctx, c.GcsAuth.TokenUrl, c.GcsAuth.ReuseTokenFromUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create token source: %w", err)
	}

	clientOpts := []option.ClientOption{option.WithTokenSource(tokenSrc)}
	options := append(baseOptions, cloudmetric.WithMonitoringClientOptions(clientOpts...))

	// Try with explicit project ID if configured.
	if c.GcsConnection.BillingProject != "" {
		optionsWithProject := append(options, cloudmetric.WithProjectID(c.GcsConnection.BillingProject))
		if exporter, err := cloudmetric.New(optionsWithProject...); err == nil {
			logger.Infof("Cloud Monitoring exporter initialized with --token-url and project %s", c.GcsConnection.BillingProject)
			return exporter, nil
		}
		logger.Infof("Failed to initialize with explicit project ID, retrying without project")
	}

	// Retry without explicit project ID.
	exporter, err := cloudmetric.New(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter with --token-url: %w", err)
	}

	logger.Infof("Cloud Monitoring exporter initialized with --token-url credentials")
	return exporter, nil
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
