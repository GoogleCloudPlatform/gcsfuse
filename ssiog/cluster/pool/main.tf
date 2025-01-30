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

variable "cluster" {
  type = string
}

variable "region" {
  type = string
}

variable "zone" {
  type = string
}

variable "service_account_email" {
  type = string
  default = "default"
}

variable "machine_type" {
  type = string
}

variable "node_count" {
  type = number
}

resource "google_container_node_pool" "base" {
  name       = "base"
  location   = var.zone # zonal node pool
  cluster    = var.cluster
  node_count = var.node_count
  node_config {
    preemptible  = false
    machine_type = var.machine_type
    kubelet_config {
        cpu_cfs_quota = "false"
        pod_pids_limit = 0
    }
    gvnic {
      enabled = true
    }
    service_account = var.service_account_email
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
    workload_metadata_config {
      mode = "GKE_METADATA"
    }
  }
}