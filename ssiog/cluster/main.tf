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
# This creates a GKE cluster with a (non-default) node pool, and some related
# resource.

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
  region  = var.region
  zone    = var.zone
}

provider "google-beta" {
  project = var.project
  region  = var.region
  zone    = var.zone
}

# We need a GCP service account that will serve as the cluster's SA.
# module "sa" {
#   source  = "./service-account"
#   project = var.project
# }

# Create the GKE cluster.
module "gke" {
  source     = "./gke"
  project    = var.project
  region     = var.region
  zone = var.zone
  cluster_name = var.cluster_name
  # depends_on = [module.sa]
}

# Retrieve an access token as the Terraform runner
data "google_client_config" "provider" {}
data "google_container_cluster" "cluster" {
  name       = module.gke.cluster_name
  location   = var.zone
  depends_on = [module.gke]
}

provider "kubernetes" {
  host  = "https://${data.google_container_cluster.princer-ssiog.endpoint}"
  token = data.google_client_config.provider.access_token
  cluster_ca_certificate = base64decode(
    data.google_container_cluster.cluster.master_auth[0].cluster_ca_certificate,
  )
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "gke-gcloud-auth-plugin"
  }
}

# Create the node pool used by the service.
module "pool" {
  source                = "./pool"
  cluster               = module.gke.cluster_name
  machine_type = var.machine_type
  node_count = var.node_count
  region                = var.region
  # service_account_email = module.sa.email
  depends_on            = [module.gke]
  zone                  = var.zone
}

# We create an artifact registry Docker repository to host any docker images.
module "registry" {
  source = "./registry"
  region = var.region
  repository_id = var.repository_id
}