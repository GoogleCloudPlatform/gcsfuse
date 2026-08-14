#!/bin/bash
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Uninstalls the Cloud Storage FUSE CSI driver from a target GKE cluster.
# Usage: tools/scripts/uninstall_csi.sh [CLUSTER_PROJECT] [CLUSTER_NAME] [CLUSTER_LOCATION] [CSI_VERSION]

set -euo pipefail

CLUSTER_PROJECT="${1:-}"
CLUSTER_NAME="${2:-}"
CLUSTER_LOCATION="${3:-}"
CSI_VERSION="${4:-main}"

if [[ -z "${CLUSTER_PROJECT}" || -z "${CLUSTER_NAME}" || -z "${CLUSTER_LOCATION}" ]]; then
  echo "Error: Target GKE cluster configuration is missing."
  echo "Please connect to a GKE cluster context or supply the variables directly:"
  echo "  gcloud container clusters get-credentials <CLUSTER_NAME> --location <LOCATION> --project <PROJECT>"
  echo "  or: make uninstall-csi CLUSTER_PROJECT=<PROJECT> CLUSTER_NAME=<CLUSTER_NAME> CLUSTER_LOCATION=<LOCATION>"
  exit 1
fi

echo "--------------------------------------"
echo "Uninstalling CSI Driver from the cluster"
echo "Target Project:  ${CLUSTER_PROJECT}"
echo "Target Cluster:  ${CLUSTER_NAME}"
echo "Target Location: ${CLUSTER_LOCATION}"
echo "CSI Version:     ${CSI_VERSION}"
echo "--------------------------------------"

read -r -p "Are you sure? (y/N): " confirm
case "$(echo "${confirm}" | tr '[:upper:]' '[:lower:]')" in
  y|yes)
    ;;
  *)
    echo "Uninstall cancelled."
    exit 0
    ;;
esac

echo "Getting cluster credentials..."
gcloud container clusters get-credentials "${CLUSTER_NAME}" --location "${CLUSTER_LOCATION}" --project "${CLUSTER_PROJECT}"

echo "Checking CSI driver installation status on cluster..."
MANAGED_ADDON=$(gcloud container clusters describe "${CLUSTER_NAME}" \
  --location "${CLUSTER_LOCATION}" \
  --project "${CLUSTER_PROJECT}" \
  --format="value(addonsConfig.gcsFuseCsiDriverConfig.enabled)" 2>/dev/null || true)

if [[ "${MANAGED_ADDON}" == "True" ]]; then
  echo "The Cloud Storage FUSE CSI driver on cluster '${CLUSTER_NAME}' is managed by GKE."
  echo "To uninstall the managed add-on, disable it via gcloud:"
  echo "  gcloud container clusters update ${CLUSTER_NAME} --location=${CLUSTER_LOCATION} --project=${CLUSTER_PROJECT} --no-enable-gcs-fuse-csi-driver"
  exit 1
fi

if ! kubectl get csidriver gcsfuse.csi.storage.gke.io >/dev/null 2>&1; then
  echo "Cloud Storage FUSE CSI driver is not installed on cluster '${CLUSTER_NAME}'. Nothing to uninstall."
  exit 0
fi

# Create temporary directory and ensure cleanup on exit
TMP_DIR=$(mktemp -d /tmp/make-uninstall-csi.XXXXXXXXXX)
trap 'rm -rf "${TMP_DIR}"' EXIT INT TERM

echo "Cloning CSI driver repository (${CSI_VERSION})..."
git clone --branch "${CSI_VERSION}" --depth 1 https://github.com/GoogleCloudPlatform/gcs-fuse-csi-driver.git "${TMP_DIR}"

echo "Uninstalling CSI driver from cluster..."
make -C "${TMP_DIR}" uninstall || echo "Warning: Uninstall encountered errors (likely already deleted resources). Proceeding..."

echo "CSI driver uninstallation completed."
