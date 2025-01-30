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

variable "region" {
  type = string
}

variable "zone" {
  type = string
}

variable "cluster_name" {
  type = string
}

resource "google_container_cluster" "princer-ssiog" {
  provider = google-beta # For secret_manager_config
  name     = var.cluster_name
  location = var.zone # Zonal cluster
  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1
  workload_identity_config {
    workload_pool = "${var.project}.svc.id.goog"
  }
  # Enable GCS Fuse in the cluster.  This does not necessarily mean we use
  # GCS Fuse to access GCS, or that every pool uses it. It does make it easier
  # to save benchmark results to GCS.
  addons_config {
    gcs_fuse_csi_driver_config {
      enabled = true
    }
  }
  secret_manager_config {
    enabled = true
  }
}

output "cluster_name" {
  value = google_container_cluster.princer-ssiog.name
}