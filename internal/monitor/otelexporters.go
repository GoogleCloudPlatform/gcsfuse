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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"cloud.google.com/go/compute/metadata"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	cloudmetric "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	"github.com/googlecloudplatform/gcsfuse/v3/cfg"
	"github.com/googlecloudplatform/gcsfuse/v3/common"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/auth"
	"github.com/googlecloudplatform/gcsfuse/v3/internal/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	globalLog "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
)

const serviceName = "gcsfuse"
const cloudMonitoringMetricPrefix = "custom.googleapis.com/gcsfuse/"

var allowedMetricPrefixes = []string{"fs/", "gcs/", "file_cache/", "buffered_read/", "grpc.", "read/"}

// SetupOTelMetricExporters sets up the metrics exporters
func SetupOTelMetricExporters(ctx context.Context, c *cfg.Config, mountID string) (shutdownFn common.ShutdownFn) {
	var shutdownFns []common.ShutdownFn
	options := make([]metric.Option, 0)

	opts, promShutdownFn := setupPrometheus(c.Metrics.PrometheusPort)
	options = append(options, opts...)
	shutdownFns = append(shutdownFns, promShutdownFn)

	projectID := ""
	if c.Metrics.ExperimentalEnableOtelMetrics {
		otelOpts, err := setupOtelMetricsEndpoint(ctx, c.Metrics.ExperimentalOtelMetricsEndpoint, c.GcsAuth, c.Metrics.CloudMetricsExportIntervalSecs)
		if err != nil {
			logger.Errorf("Error while creating OTLP metric exporter: %v", err)
		} else {
			options = append(options, otelOpts...)
		}
		projectID = getProjectID(ctx, c.GcsAuth, c.Metrics.ExperimentalOtelMetricsProjectId)
	} else {
		opts := setupCloudMonitoring(c.Metrics.CloudMetricsExportIntervalSecs)
		options = append(options, opts...)
	}
	res, err := getOtelResource(ctx, mountID, projectID)
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
	// If we get a PermissionDenied error (gRPC) or 403 Forbidden (HTTP), disable the exporter to prevent future attempts.
	if err != nil && isPermissionDenied(err) {
		if e.disabled.CompareAndSwap(false, true) {
			logger.Errorf("Disabling metrics exporter due to permission denied error: %v", err)
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

// permissionAwareLogExporter wraps a log.Exporter and disables itself if it encounters
// a PermissionDenied error. This prevents log spam when the environment lacks
// necessary permissions.
type permissionAwareLogExporter struct {
	log.Exporter
	// disabled indicates whether the exporter has been permanently disabled.
	disabled atomic.Bool
}

func (e *permissionAwareLogExporter) Export(ctx context.Context, records []log.Record) error {
	// Check if disabled before attempting export to save resources and avoid noise.
	if e.disabled.Load() {
		return nil
	}

	for i := range records {
		// Optimize performance by using a fast integer comparison instead of a string operation
		// or rewriting all logs. LevelTrace (-8) maps to SeverityTrace1 (1).
		if int(records[i].Severity()) == 1 /* otellog.SeverityTrace1 */ {
			records[i].SetSeverityText("TRACE")
		}
	}

	err := e.Exporter.Export(ctx, records)
	// If we get a PermissionDenied error (gRPC) or 403 Forbidden (HTTP), disable the exporter to prevent future attempts.
	if err != nil && isPermissionDenied(err) {
		if e.disabled.CompareAndSwap(false, true) {
			logger.Errorf("Disabling Cloud Logging exporter due to permission denied error: %v", err)
		}
	}
	return err
}

func (e *permissionAwareLogExporter) ForceFlush(ctx context.Context) error {
	if e.disabled.Load() {
		return nil
	}
	return e.Exporter.ForceFlush(ctx)
}

// isPermissionDenied checks whether the error represents a permission denied or HTTP 403 Forbidden error.
// It avoids matching generic strings containing "403" (e.g., port numbers or IP addresses).
func isPermissionDenied(err error) bool {
	if err == nil {
		return false
	}

	// 1. Check gRPC status code
	if status.Code(err) == codes.PermissionDenied {
		return true
	}

	// 2. Check typed Google API error
	var gErr *googleapi.Error
	if errors.As(err, &gErr) && gErr.Code == http.StatusForbidden {
		return true
	}

	// 3. Check specific HTTP 403 string patterns (e.g., OTLP HTTP exporter errors)
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "403 forbidden") ||
		strings.Contains(msg, "status code: 403") ||
		strings.Contains(msg, "status code 403") ||
		strings.Contains(msg, "googleapi: error 403") ||
		strings.Contains(msg, "\"code\": 403") ||
		strings.Contains(msg, "\"code\":403")
}

func metricFormatter(m metricdata.Metrics) string {
	return cloudMonitoringMetricPrefix + strings.ReplaceAll(m.Name, ".", "/")
}

func setupOtelMetricsEndpoint(ctx context.Context, endpoint string, authConfig cfg.GcsAuthConfig, intervalSecs int64) ([]metric.Option, error) {
	if intervalSecs <= 0 {
		return nil, nil
	}
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
	}

	if isOtelEndpointInsecure(endpoint) {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	if strings.Contains(endpoint, "googleapis.com") {
		client, err := getGcpOtelHttpClient(ctx, authConfig, "https://www.googleapis.com/auth/monitoring.write")
		if err != nil {
			logger.Errorf("Error getting GCP authenticated token source for metrics: %v", err)
			return nil, err
		}
		opts = append(opts, otlpmetrichttp.WithHTTPClient(client))
	}

	exporter, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// Wrap the exporter to handle permission denied errors
	wrappedExporter := &permissionAwareExporter{
		Exporter: exporter,
	}

	reader := metric.NewPeriodicReader(wrappedExporter, metric.WithInterval(time.Duration(intervalSecs)*time.Second))
	return []metric.Option{metric.WithReader(reader)}, nil
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

func getProjectID(ctx context.Context, authConfig cfg.GcsAuthConfig, configuredProjectID string) string {
	if configuredProjectID != "" {
		return configuredProjectID
	}

	if authConfig.KeyFile != "" {
		contents, err := os.ReadFile(string(authConfig.KeyFile))
		if err != nil {
			logger.Errorf("Failed to read key file for project ID: %v", err)
		} else {
			var config struct {
				ProjectID string `json:"project_id"`
			}
			if err := json.Unmarshal(contents, &config); err != nil {
				logger.Errorf("Failed to parse key file for project ID: %v", err)
			} else if config.ProjectID != "" {
				return config.ProjectID
			}
		}
	} else if authConfig.TokenUrl == "" {
		creds, err := google.FindDefaultCredentials(ctx)
		if err == nil && creds.ProjectID != "" {
			return creds.ProjectID
		}
	}

	if envProj := os.Getenv("GOOGLE_CLOUD_PROJECT"); envProj != "" {
		return envProj
	}

	if metadata.OnGCE() {
		if proj, err := metadata.ProjectIDWithContext(ctx); err == nil && proj != "" {
			return proj
		}
	}
	return ""
}

// SetupOTelLogExporter initializes the OpenTelemetry Log provider.
func SetupOTelLogExporter(ctx context.Context, endpoint string, mountID string, authConfig cfg.GcsAuthConfig, configuredProjectID string) (common.ShutdownFn, error) {
	projectID := getProjectID(ctx, authConfig, configuredProjectID)
	res, err := getOtelResource(ctx, mountID, projectID)
	if err != nil {
		logger.Errorf("Error while fetching resource for logs: %v", err)
		return nil, err
	}

	opts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(endpoint),
		otlploghttp.WithCompression(otlploghttp.GzipCompression),
	}

	if isOtelEndpointInsecure(endpoint) {
		opts = append(opts, otlploghttp.WithInsecure())
	}

	if strings.Contains(endpoint, "googleapis.com") {
		client, err := getGcpOtelHttpClient(ctx, authConfig, "https://www.googleapis.com/auth/logging.write")
		if err != nil {
			logger.Errorf("Error getting GCP authenticated token source for logs: %v", err)
			return nil, err
		}
		opts = append(opts, otlploghttp.WithHTTPClient(client))
	}

	exporter, err := otlploghttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// Wrap the exporter to handle permission denied errors
	wrappedExporter := &permissionAwareLogExporter{
		Exporter: exporter,
	}

	processor := log.NewBatchProcessor(
		wrappedExporter,
		log.WithExportMaxBatchSize(2000),
		log.WithMaxQueueSize(8192),
		log.WithExportInterval(5*time.Second),
		log.WithExportTimeout(10*time.Second),
	)
	provider := log.NewLoggerProvider(
		log.WithProcessor(processor),
		log.WithResource(res),
	)

	// Optional: set it globally if needed
	globalLog.SetLoggerProvider(provider)

	return func(ctx context.Context) error {
		return provider.Shutdown(ctx)
	}, nil
}

// getOtelResource creates a base OTel resource and merges the GCP project ID if provided.
func getOtelResource(ctx context.Context, mountID string, projectID string) (*resource.Resource, error) {
	res, err := getResource(ctx, mountID)
	if err != nil {
		logger.Errorf("Error while fetching resource: %v", err)
		if res == nil {
			return nil, err
		}
	}
	if projectID != "" {
		projRes, _ := resource.New(ctx,
			resource.WithSchemaURL(res.SchemaURL()),
			resource.WithAttributes(
				attribute.String("gcp.project_id", projectID),
			),
		)
		if mergedRes, err := resource.Merge(res, projRes); err != nil {
			logger.Errorf("Error merging project ID into resource: %v", err)
		} else {
			res = mergedRes
		}
	}
	return res, nil
}

// isOtelEndpointInsecure returns true if the endpoint is determined to be a local testing endpoint.
func isOtelEndpointInsecure(endpoint string) bool {
	return strings.Contains(endpoint, "localhost") || strings.Contains(endpoint, "127.0.0.1") || strings.Contains(endpoint, "0.0.0.0") || strings.Contains(endpoint, "[::1]")
}

// getGcpOtelHttpClient creates an authenticated HTTP client for a specific GCP scope.
func getGcpOtelHttpClient(ctx context.Context, authConfig cfg.GcsAuthConfig, scope string) (*http.Client, error) {
	ts, err := auth.GetTokenSourceWithScope(ctx, string(authConfig.KeyFile), authConfig.TokenUrl, authConfig.ReuseTokenFromUrl, scope)
	if err != nil {
		return nil, err
	}
	client := oauth2.NewClient(ctx, ts)
	client.Timeout = 30 * time.Second
	return client, nil
}
