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


# Create a random string to uniquely name per-project or global resources.
resource "random_id" "uniq" {
  byte_length = 8
}

resource "google_storage_bucket_iam_member" "grant-ksa-permissions-on-data-bucket" {
  bucket = var.bucket_name
  role   = "roles/storage.objectUser"
  member = "principal:${var.k8s_sa_full}"
}

locals {
  # This is a placeholder value. There is no relation to the GCS storage class.
  storage_class = "gcsfuse-sc"
  pv_name       = "data-bucket-pv-${random_id.uniq.hex}"
  pvc_name      = "data-bucket-pvc-${random_id.uniq.hex}"
}

# Use Static provisioning to define a PV for GCSFuse:
#   https://cloud.google.com/kubernetes-engine/docs/how-to/persistent-volumes/cloud-storage-fuse-csi-driver#provision-static.
# For other storage solutions, please define your own PV and PVC and prefill the volume with the datasets.
resource "kubernetes_persistent_volume" "data-bucket-pv" {
  metadata {
    name = local.pv_name
  }
  spec {
    access_modes = ["ReadWriteMany"]
    capacity = {
      storage = "64Gi"
    }
    persistent_volume_reclaim_policy = "Retain"
    storage_class_name               = local.storage_class
    claim_ref {
      name = local.pvc_name
    }
    persistent_volume_source {
      csi {
        driver        = "gcsfuse.csi.storage.gke.io"
        volume_handle = var.bucket_name
        volume_attributes = {
          "enable_metrics" : "true"
          "skip_sci_bucket_access_check" : "true"
        }
      }
    }
    mount_options = [
      "debug_fuse",
      # "implicit-dirs", #avoid if possible
      "max-conns-per-host=0",
      "metadata-cache:ttl-secs:-1",
      "metadata-cache:stat-cache-max-size-mb:-1",
      "metadata-cache:type-cache-max-size-mb:-1",
      "file-system:kernel-list-cache-ttl-secs:-1",
      "file-cache:max-size-mb:-1",
      "file-cache:cache-file-for-range-read:true",
      "file-cache:enable-parallel-downloads:true",
    ]
  }
}

resource "kubernetes_persistent_volume_claim" "pvc" {
  metadata {
    name = local.pvc_name
  }
  spec {
    access_modes = ["ReadWriteMany"]
    resources {
      requests = {
        storage = "64Gi"
      }
    }
    volume_name        = local.pv_name
    storage_class_name = local.storage_class
  }
  depends_on = [kubernetes_persistent_volume.data-bucket-pv]
}
