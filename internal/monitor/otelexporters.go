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
	"time"

	cloudmetric "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const serviceName = "gcsfuse"

// SetupOTelMetricExporters sets up the metrics exporters
func SetupOTelMetricExporters(ctx context.Context, c *cfg.Config) (shutdownFn common.ShutdownFn) {
	shutdownFns := make([]common.ShutdownFn, 0)
	options := make([]metric.Option, 0)

	opts, shutdownFn := setupPrometheus(c.Metrics.PrometheusPort)
	options = append(options, opts...)
	shutdownFns = append(shutdownFns, shutdownFn)

	opts, shutdownFn = setupCloudMonitoring(c.Metrics.CloudMetricsExportIntervalSecs)
	options = append(options, opts...)
	shutdownFns = append(shutdownFns, shutdownFn)

	res, err := getResource(ctx)
	if err != nil {
		logger.Errorf("Error while fetching resource: %v", err)
	} else {
		options = append(options, metric.WithResource(res))
	}

	meterProvider := metric.NewMeterProvider(options...)
	shutdownFns = append(shutdownFns, meterProvider.Shutdown)

	otel.SetMeterProvider(meterProvider)

	return common.JoinShutdownFunc(shutdownFns...)
}

func setupCloudMonitoring(secs int64) ([]metric.Option, common.ShutdownFn) {
	if secs <= 0 {
		return nil, nil
	}
	options := []cloudmetric.Option{
		cloudmetric.WithMetricDescriptorTypeFormatter(metricFormatter),
		cloudmetric.WithFilteredResourceAttributes(func(kv attribute.KeyValue) bool {
			// Ensure that PID is available as a metric label on metrics explorer.
			return cloudmetric.DefaultResourceAttributesFilter(kv) ||
				kv.Key == semconv.ProcessPIDKey
		}),
	}
	exporter, err := cloudmetric.New(options...)
	if err != nil {
		logger.Errorf("Error while creating Google Cloud exporter:%v", err)
		return nil, nil
	}

	r := metric.NewPeriodicReader(exporter, metric.WithInterval(time.Duration(secs)*time.Second))
	return []metric.Option{metric.WithReader(r)}, r.Shutdown
}

func metricFormatter(m metricdata.Metrics) string {
	return "custom.googleapis.com/gcsfuse/" + strings.ReplaceAll(m.Name, ".", "/")
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
	done := make(chan interface{})
	go serveMetrics(port, shutdownCh, done)
	return []metric.Option{metric.WithReader(exporter)}, func(ctx context.Context) error {
		shutdownCh <- ctx
		close(shutdownCh)
		<-done
		close(done)
		return nil
	}
}

func serveMetrics(port int64, shutdownCh <-chan context.Context, done chan<- interface{}) {
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

func getResource(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx,
		// Use the GCP resource detector to detect information about the GCP platform
		resource.WithDetectors(gcp.NewDetector()),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(common.GetVersion()),
		),
	)
}
