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

# Builds and installs the Cloud Storage FUSE CSI driver onto a target GKE cluster.
# Usage: tools/scripts/install_csi.sh [CLUSTER_PROJECT] [CLUSTER_NAME] [CLUSTER_LOCATION] [GCSFUSE_TAG] [CSI_VERSION] [OVERLAY]

set -euo pipefail

CLUSTER_PROJECT="${1:-}"
CLUSTER_NAME="${2:-}"
CLUSTER_LOCATION="${3:-}"
GCSFUSE_TAG="${4:-master}"
CSI_VERSION="${5:-main}"
OVERLAY="${6:-stable}"

if [[ -z "${CLUSTER_PROJECT}" || -z "${CLUSTER_NAME}" || -z "${CLUSTER_LOCATION}" ]]; then
  echo "Error: Target GKE cluster configuration is missing."
  echo "Please connect to a GKE cluster context or supply the variables directly:"
  echo "  gcloud container clusters get-credentials <CLUSTER_NAME> --location <LOCATION> --project <PROJECT>"
  echo "  or: make install-csi CLUSTER_PROJECT=<PROJECT> CLUSTER_NAME=<CLUSTER_NAME> CLUSTER_LOCATION=<LOCATION>"
  exit 1
fi

# Automatically compute REGISTRY based on cluster location and project
REGISTRY_REGION="$(echo "${CLUSTER_LOCATION}" | cut -d'-' -f1-2)"
REGISTRY="${REGISTRY_REGION}-docker.pkg.dev/${CLUSTER_PROJECT}/csi-dev"

# Automatically compute STAGINGVERSION
CSI_COMMIT_SHA=$(git ls-remote https://github.com/GoogleCloudPlatform/gcs-fuse-csi-driver.git "${CSI_VERSION}" 2>/dev/null | head -n 1 | cut -c1-7)
if [[ -z "${CSI_COMMIT_SHA}" ]]; then
  CSI_COMMIT_SHA="${CSI_VERSION:0:7}"
fi

GCSFUSE_COMMIT_SHA=$(git ls-remote https://github.com/GoogleCloudPlatform/gcsfuse.git "${GCSFUSE_TAG}" 2>/dev/null | head -n 1 | cut -c1-7)
if [[ -z "${GCSFUSE_COMMIT_SHA}" ]]; then
  GCSFUSE_COMMIT_SHA="${GCSFUSE_TAG:0:7}"
fi

STAGINGVERSION="prow-gob-internal-boskos-csi-${CSI_COMMIT_SHA}-fuse-${GCSFUSE_COMMIT_SHA}"

echo "--------------------------------------"
echo "Installing CSI Driver onto the cluster"
echo "Target Project:  ${CLUSTER_PROJECT}"
echo "Target Cluster:  ${CLUSTER_NAME}"
echo "Target Location: ${CLUSTER_LOCATION}"
echo "Staging Version: ${STAGINGVERSION}"
echo "Registry:        ${REGISTRY}"
echo "Overlay:         ${OVERLAY}"
echo "CSI Version:     ${CSI_VERSION}"
echo "GCSFuse Tag:     ${GCSFUSE_TAG}"
echo "--------------------------------------"

# Prompt for confirmation if running in an interactive terminal
if [[ -t 0 ]]; then
  read -r -p "Are you sure? (y/N): " confirm
  case "$(echo "${confirm}" | tr '[:upper:]' '[:lower:]')" in
    y|yes)
      ;;
    *)
      echo "Installation cancelled."
      exit 0
      ;;
  esac
fi

echo "Getting cluster credentials..."
gcloud container clusters get-credentials "${CLUSTER_NAME}" --location "${CLUSTER_LOCATION}" --project "${CLUSTER_PROJECT}"

echo "Checking if CSI driver is already installed on the cluster..."
MANAGED_ADDON=$(gcloud container clusters describe "${CLUSTER_NAME}" \
  --location "${CLUSTER_LOCATION}" \
  --project "${CLUSTER_PROJECT}" \
  --format="value(addonsConfig.gcsFuseCsiDriverConfig.enabled)" 2>/dev/null || true)

if [[ "${MANAGED_ADDON}" == "True" ]]; then
  echo "Error: GKE Managed Cloud Storage FUSE CSI driver add-on is currently enabled on cluster '${CLUSTER_NAME}'."
  echo "Please disable the managed add-on first before installing a custom driver:"
  echo "  gcloud container clusters update ${CLUSTER_NAME} --location=${CLUSTER_LOCATION} --project=${CLUSTER_PROJECT} --no-enable-gcs-fuse-csi-driver"
  exit 1
fi

if kubectl get csidriver gcsfuse.csi.storage.gke.io >/dev/null 2>&1; then
  echo "Error: Cloud Storage FUSE CSI driver is already installed on cluster '${CLUSTER_NAME}'."
  echo "Please uninstall the existing driver first by running: make uninstall-csi"
  exit 1
fi

# Create temporary directory and ensure cleanup on exit
TMP_DIR=$(mktemp -d /tmp/make-install-csi.XXXXXXXXXX)
trap 'rm -rf "${TMP_DIR}"' EXIT INT TERM

echo "Cloning CSI driver repository (${CSI_VERSION})..."
git clone --branch "${CSI_VERSION}" --depth 1 https://github.com/GoogleCloudPlatform/gcs-fuse-csi-driver.git "${TMP_DIR}"

echo "Configuring docker authentication for ${REGISTRY_REGION}-docker.pkg.dev..."
gcloud auth configure-docker "${REGISTRY_REGION}-docker.pkg.dev" --quiet

echo "Building and pushing multi-arch image..."
make -C "${TMP_DIR}" build-image-and-push-multi-arch \
  REGISTRY="${REGISTRY}" \
  STAGINGVERSION="${STAGINGVERSION}" \
  BUILD_GCSFUSE_FROM_SOURCE=true \
  GCSFUSE_TAG="${GCSFUSE_TAG}"

echo "Installing image to cluster..."
make -C "${TMP_DIR}" install \
  PROJECT="${CLUSTER_PROJECT}" \
  CLUSTER_NAME="${CLUSTER_NAME}" \
  CLUSTER_LOCATION="${CLUSTER_LOCATION}" \
  REGISTRY="${REGISTRY}" \
  STAGINGVERSION="${STAGINGVERSION}" \
  OVERLAY="${OVERLAY}"

echo ""
echo "--------------------------------------"
echo "CSI driver installation completed successfully."
echo ""
echo "Installed Images Summary:"
echo "  Driver:            ${REGISTRY}/gcs-fuse-csi-driver:${STAGINGVERSION}"
echo "  Sidecar Mounter:   ${REGISTRY}/gcs-fuse-csi-driver-sidecar-mounter:${STAGINGVERSION}"
echo "  Webhook:           ${REGISTRY}/gcs-fuse-csi-driver-webhook:${STAGINGVERSION}"
echo "  Metadata Prefetch: ${REGISTRY}/gcs-fuse-csi-driver-metadata-prefetch:${STAGINGVERSION}"
echo "--------------------------------------"
