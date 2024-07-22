// Copyright 2021 Google Inc. All Rights Reserved.
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
	"net/http"
	"strings"
	"time"

	"contrib.go.opencensus.io/exporter/ocagent"
	"contrib.go.opencensus.io/exporter/prometheus"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"go.opencensus.io/stats/view"
)

var stackdriverExporter *stackdriver.Exporter

// EnableStackdriverExporter starts to collect monitoring metrics and exports
// them to Stackdriver iff the given interval is positive.
func EnableStackdriverExporter(interval time.Duration) error {
	if interval <= 0 {
		return nil
	}

	var err error
	if stackdriverExporter, err = stackdriver.NewExporter(stackdriver.Options{
		ReportingInterval: interval,
		OnError: func(err error) {
			logger.Errorf("Fail to send metric: %v", err)
		},

		// For a local metric "http_sent_bytes", the Stackdriver metric type
		// would be "custom.googleapis.com/gcsfuse/http_sent_bytes", display
		// name would be "Http sent bytes".
		MetricPrefix: "custom.googleapis.com/gcsfuse/",
		GetMetricDisplayName: func(view *view.View) string {
			name := strings.ReplaceAll(view.Name, "_", " ")
			if len(name) > 0 {
				name = strings.ToUpper(name[:1]) + name[1:]
			}
			return name
		},
	}); err != nil {
		return fmt.Errorf("create stackdriver exporter: %w", err)
	}
	if err = stackdriverExporter.StartMetricsExporter(); err != nil {
		return fmt.Errorf("start stackdriver exporter: %w", err)
	}

	logger.Info("Stackdriver exporter started")
	return nil
}

// CloseStackdriverExporter ensures all collected metrics are sent to
// Stackdriver and closes the stackdriverExporter.
func CloseStackdriverExporter() {
	if stackdriverExporter != nil {
		stackdriverExporter.StopMetricsExporter()
		stackdriverExporter.Flush()
	}
	stackdriverExporter = nil
}

var ocExporter *ocagent.Exporter

// EnableOpenTelemetryCollectorExporter starts exporting monitoring metrics to
// the OpenTelemetry Collector at the given address.
// Details: https://opentelemetry.io/docs/collector/
func EnableOpenTelemetryCollectorExporter(address string) error {
	if address == "" {
		return nil
	}

	var err error
	if ocExporter, err = ocagent.NewExporter(
		ocagent.WithAddress(address),
		ocagent.WithServiceName("gcsfuse"),
		ocagent.WithReconnectionPeriod(5*time.Second),
	); err != nil {
		return fmt.Errorf("create opentelementry collector exporter: %w", err)
	}

	view.RegisterExporter(ocExporter)
	logger.Info("OpenTelemetry collector exporter started")
	return nil
}

// CloseOpenTelemetryCollectorExporter ensures all collected metrics are sent to
// the OpenTelemetry Collect and closes the exporter.
func CloseOpenTelemetryCollectorExporter() {
	if ocExporter != nil {
		ocExporter.Stop()
		ocExporter.Flush()
	}
	ocExporter = nil
}

var prometheusExporter *prometheus.Exporter
var prometheusServer *http.Server

// EnablePrometheusCollectorExporter starts exporting monitoring metrics for
// the Prometheus to scrape on the given port.
func EnablePrometheusCollectorExporter(port int) error {
	if port == 0 {
		return nil
	}

	if prometheusServer != nil {
		return errors.New("a Prometheus server is already running")
	}

	if prometheusExporter != nil {
		return errors.New("a Prometheus exporter is already running")
	}

	var err error
	if prometheusExporter, err = prometheus.NewExporter(
		prometheus.Options{
			OnError: func(err error) {
				logger.Errorf("Fail to collect metric: %v", err)
			},
		},
	); err != nil {
		return fmt.Errorf("create Prometheus collector exporter: %w", err)
	}

	view.RegisterExporter(prometheusExporter)

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", prometheusExporter.ServeHTTP)
	prometheusServer = &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := prometheusServer.ListenAndServe(); err != nil {
			logger.Errorf("Failed to start Prometheus server: %v", err)
		}
	}()

	logger.Info("Prometheus collector exporter started")
	return nil
}

// ClosePrometheusCollectorExporter closes the Prometheus exporter.
func ClosePrometheusCollectorExporter() {
	if prometheusServer != nil {
		if err := prometheusServer.Shutdown(context.Background()); err != nil {
			logger.Errorf("Failed to shutdown Prometheus server: %v", err)
		}
	}

	if prometheusExporter != nil {
		view.UnregisterExporter(prometheusExporter)
	}

	prometheusServer = nil
	prometheusExporter = nil
}
