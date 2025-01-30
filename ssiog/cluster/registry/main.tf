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

variable "region" {
  type = string
}

variable "repository_id" {
  type = string
}

resource "google_artifact_registry_repository" "ssiog-registry" {
  location      = var.region
  repository_id = var.repository_id
  description   = "Docker repository for ssiog benchmarks"
  format        = "DOCKER"
}