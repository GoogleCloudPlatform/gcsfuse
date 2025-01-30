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

terraform {
  required_providers {
    # This is used to create Google Cloud Platform resources.
    google = {
      source  = "hashicorp/google"
      version = ">= 5.44.1"
    }
    # This is used to create the k8s resources within the cluster created by
    # this configuration file.
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "2.31.0"
    }
  }
}

provider "google" {
  project = var.project
}

# Retrieve an access token as the Terraform runner
data "google_client_config" "provider" {}

data "google_project" "project" {}

data "google_container_cluster" "cluster" {
  name     = var.cluster_name
  location = var.location
}

provider "kubernetes" {
  host  = "https://${data.google_container_cluster.cluster.endpoint}"
  token = data.google_client_config.provider.access_token
  cluster_ca_certificate = base64decode(
    data.google_container_cluster.cluster.master_auth[0].cluster_ca_certificate,
  )
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "gke-gcloud-auth-plugin"
  }
}

# Create a random string to uniquely name per-project or global resources.
resource "random_id" "uniq" {
  byte_length = 8
}

locals {
  k8s_sa_name = var.k8s_sa_name

  # The full name of the k8s service account when used in GCP IAM bindings.
  k8s_sa_full = "//iam.googleapis.com/projects/${data.google_project.project.number}/locations/global/workloadIdentityPools/${data.google_project.project.project_id}.svc.id.goog/subject/ns/default/sa/${local.k8s_sa_name}"
}

resource "kubernetes_service_account" "ksa" {
  metadata {
    name = local.k8s_sa_name
  }
}

resource "google_storage_bucket_iam_member" "grant-ksa-permissions-on-metrics-bucket" {
  bucket     = var.metrics_bucket_name
  role       = "roles/storage.objectUser"
  member     = "principal:${local.k8s_sa_full}"
}

resource "google_storage_bucket_iam_member" "grant-ksa-permissions-on-data-bucket" {
  bucket     = var.data_bucket_name
  role       = "roles/storage.objectUser"
  member     = "principal:${local.k8s_sa_full}"
}

# Find the latest SHA of the image, this forces a pull if the image changes.
data "google_artifact_registry_docker_image" "image" {
  location      = "us-west1"
  repository_id = var.repository_id
  image_name    = var.image_name
}

locals {
  parallelism        = var.parallelism
  epochs             = var.epochs
  prefixes           = var.prefixes  # gs://${var.data_bucket_name}"
  background_threads = var.background_threads
  # This is large enough to make the L-SSD cache irrelevant
  # object_count_limit = 1024
  # This is good for troubleshooting, basically one file per thread
  object_count_limit = local.parallelism * local.background_threads
  # file_size_gib      = 2
  memory             = -1 # local.file_size_gib * local.background_threads
  label              = var.label
}

# Generate the data loader job-set (nodes talks to each other) benchmark definition.
resource "local_file" "ssiog-training-jobset" {
  filename = "job-set.yaml"
  content = templatefile("./templates/job-set.tfpl.yaml", {
    image               = data.google_artifact_registry_docker_image.image.self_link,
    prefixes            = local.prefixes
    k8s_sa_name         = local.k8s_sa_name,
    metrics_bucket_name = var.metrics_bucket_name,
    data_bucket_name    = var.data_bucket_name,
    parallelism         = local.parallelism,
    epochs              = local.epochs,
    background_threads  = local.background_threads,
    object_count_limit  = local.object_count_limit,
    memory              = local.memory,
    label               = local.label,
    steps               = var.steps,
    batch_size          = var.batch_size,
  })
}

# Generate the data loader job (every job is independent) benchmark definition.
resource "local_file" "ssiog-training-job" {
  filename = "job.yaml"
  content = templatefile("./templates/job.tfpl.yaml", {
    image               = data.google_artifact_registry_docker_image.image.self_link,
    prefixes            = local.prefixes
    k8s_sa_name         = local.k8s_sa_name,
    metrics_bucket_name = var.metrics_bucket_name,
    data_bucket_name    = var.data_bucket_name,
    parallelism         = local.parallelism,
    epochs              = local.epochs,
    background_threads  = local.background_threads,
    object_count_limit  = local.object_count_limit,
    memory              = local.memory,
    label               = local.label,
    steps               = var.steps,
    batch_size          = var.batch_size,
  })
}
