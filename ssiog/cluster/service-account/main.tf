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

variable "project" {
  type = string
}

# Create a service account for the GKE cluster. This service account will be
# granted permissions to:
# - Publish new Cloud Trace traces
# - Publish new metrics to Cloud Monitoring
# - Publish profiles to Cloud Profiler
# - Publish logs to Cloud Logging
resource "google_service_account" "sa" {
  account_id   = "ssiog-runner"
  display_name = "ssiog-runner"
}

output "email" {
  value = google_service_account.sa.email
}

# Grant the service account permissions to publish metrics, profiles, traces and
# logs on the project

resource "google_project_iam_member" "grant-sa-cloud-monitoring-permissions" {
  project    = var.project
  role       = "roles/monitoring.metricWriter"
  member     = "serviceAccount:${google_service_account.sa.email}"
  depends_on = [google_service_account.sa]
}

resource "google_project_iam_member" "grant-sa-cloud-logging-permissions" {
  project    = var.project
  role       = "roles/logging.logWriter"
  member     = "serviceAccount:${google_service_account.sa.email}"
  depends_on = [google_service_account.sa]
}

resource "google_project_iam_member" "grant-sa-artifact-registry-permissions" {
  project    = var.project
  role       = "roles/artifactregistry.reader"
  member     = "serviceAccount:${google_service_account.sa.email}"
  depends_on = [google_service_account.sa]
}
