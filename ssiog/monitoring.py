#!/usr/bin/env python3
# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from opentelemetry import metrics
from opentelemetry.exporter.cloud_monitoring import CloudMonitoringMetricsExporter
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader, ConsoleMetricExporter
from opentelemetry.sdk.resources import Resource

# Import the GCP resource detector
from opentelemetry.sdk.resources import get_aggregated_resources 
from opentelemetry.resourcedetector.gcp_resource_detector import (
    GoogleCloudResourceDetector,
)

def initialize_monitoring_provider(exporter_type="console", export_interval_millis=10000):
    """Initializes and returns a configured OpenTelemetry MeterProvider 
    for Cloud Monitoring with GCP resource detection."""

    if exporter_type == "console":
        exporter = ConsoleMetricExporter()
    elif exporter_type == "cloud":
        exporter = CloudMonitoringMetricsExporter()
    else:
        raise ValueError(f"Unsupported exporter type: {exporter_type}")

    metrics.set_meter_provider(
        MeterProvider(
            metric_readers=[
                PeriodicExportingMetricReader(
                    exporter,
                    export_interval_millis=export_interval_millis,
                )
            ],
            # TODO (raj-prince): Implement gcp resource detector.
            resource = get_aggregated_resources([GoogleCloudResourceDetector(raise_on_error=True)]
)
        )
    )

    return metrics.get_meter(__name__)
