// Copyright 2021 Google LLC
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

	"contrib.go.opencensus.io/exporter/ocagent"
	"contrib.go.opencensus.io/exporter/prometheus"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/common"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"go.opencensus.io/stats/view"
)

func SetupOpenCensusExporters(c *cfg.Config) common.ShutdownFn {
	var shutdownFuncs []common.ShutdownFn

	if c.Metrics.ExportIntervalSecs > 0 {
		stackdriverExporter, err := enableStackdriverExporter(time.Duration(c.Metrics.ExportIntervalSecs) * time.Second)
		if err != nil {
			logger.Errorf("Unable to start stackdriver exporter: %v", err)
		} else {
			shutdownFuncs = append(shutdownFuncs, func(_ context.Context) error {
				closeStackdriverExporter(stackdriverExporter)
				return nil
			})
		}
	}
	if c.Metrics.PrometheusPort > 0 {
		exporter, server, err := enablePrometheusCollectorExporter(c.Metrics.PrometheusPort)
		if err != nil {
			logger.Errorf("Unable to start Prometheus exporter: %v", err)
		} else {
			shutdownFuncs = append(shutdownFuncs, func(_ context.Context) error {
				closePrometheusCollectorExporter(exporter, server)
				return nil
			})
		}
	}
	if c.Monitoring.ExperimentalOpentelemetryCollectorAddress != "" {
		ocExporter, err := enableOpenTelemetryCollectorExporter(c.Monitoring.ExperimentalOpentelemetryCollectorAddress)
		if err != nil {
			logger.Errorf("Unable to start OC Agent exporter: %v", err)
		} else {
			shutdownFuncs = append(shutdownFuncs, func(_ context.Context) error {
				closeOpenTelemetryCollectorExporter(ocExporter)
				return nil
			})
		}
	}
	return common.JoinShutdownFunc(shutdownFuncs...)
}

// enableStackdriverExporter starts to collect monitoring metrics and exports
// them to Stackdriver iff the given interval is positive.
func enableStackdriverExporter(interval time.Duration) (*stackdriver.Exporter, error) {
	var err error
	var stackdriverExporter *stackdriver.Exporter
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
		return nil, fmt.Errorf("create stackdriver exporter: %w", err)
	}
	if err = stackdriverExporter.StartMetricsExporter(); err != nil {
		return nil, fmt.Errorf("start stackdriver exporter: %w", err)
	}

	logger.Info("Stackdriver exporter started")
	return stackdriverExporter, nil
}

// closeStackdriverExporter ensures all collected metrics are sent to
// Stackdriver and closes the stackdriverExporter.
func closeStackdriverExporter(stackdriverExporter *stackdriver.Exporter) {
	if stackdriverExporter != nil {
		stackdriverExporter.StopMetricsExporter()
		stackdriverExporter.Flush()
	}
	stackdriverExporter = nil
}

// enableOpenTelemetryCollectorExporter starts exporting monitoring metrics to
// the OpenTelemetry Collector at the given address.
// Details: https://opentelemetry.io/docs/collector/
func enableOpenTelemetryCollectorExporter(address string) (*ocagent.Exporter, error) {
	ocExporter, err := ocagent.NewExporter(
		ocagent.WithAddress(address),
		ocagent.WithServiceName("gcsfuse"),
		ocagent.WithReconnectionPeriod(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("create opentelementry collector exporter: %w", err)
	}

	view.RegisterExporter(ocExporter)
	logger.Info("OpenTelemetry collector exporter started")
	return ocExporter, nil
}

// closeOpenTelemetryCollectorExporter ensures all collected metrics are sent to
// the OpenTelemetry Collect and closes the exporter.
func closeOpenTelemetryCollectorExporter(ocExporter *ocagent.Exporter) {
	ocExporter.Stop()
	ocExporter.Flush()
}

// enablePrometheusCollectorExporter starts exporting monitoring metrics for
// the Prometheus to scrape on the given port.
func enablePrometheusCollectorExporter(port int64) (*prometheus.Exporter, *http.Server, error) {
	prometheusExporter, err := prometheus.NewExporter(
		prometheus.Options{
			OnError: func(err error) {
				logger.Errorf("Fail to collect metric: %v", err)
			},
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create Prometheus collector exporter: %w", err)
	}

	view.RegisterExporter(prometheusExporter)

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", prometheusExporter.ServeHTTP)
	prometheusServer := &http.Server{
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
	return prometheusExporter, prometheusServer, nil
}

// closePrometheusCollectorExporter closes the Prometheus exporter.
func closePrometheusCollectorExporter(prometheusExporter *prometheus.Exporter, prometheusServer *http.Server) {
	if prometheusServer != nil {
		if err := prometheusServer.Shutdown(context.Background()); err != nil {
			logger.Errorf("Failed to shutdown Prometheus server: %v", err)
		}
	}

	if prometheusExporter != nil {
		view.UnregisterExporter(prometheusExporter)
	}
}
