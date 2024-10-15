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
	"fmt"
	"time"

	"contrib.go.opencensus.io/exporter/ocagent"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"go.opencensus.io/stats/view"
)

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
