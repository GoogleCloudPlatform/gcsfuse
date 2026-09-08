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

# Uninstalls/disables the GKE managed Cloud Storage FUSE CSI driver on a target GKE cluster.
# Usage: tools/scripts/csi/uninstall_managed.sh [CLUSTER_PROJECT] [CLUSTER_NAME] [CLUSTER_LOCATION]

set -euo pipefail

CLUSTER_PROJECT="${1:-}"
CLUSTER_NAME="${2:-}"
CLUSTER_LOCATION="${3:-}"

if [[ -z "${CLUSTER_PROJECT}" || -z "${CLUSTER_NAME}" || -z "${CLUSTER_LOCATION}" ]]; then
  echo "Error: Target GKE cluster configuration is missing."
  echo "Please connect to a GKE cluster context or supply the variables directly:"
  echo "  gcloud container clusters get-credentials <CLUSTER_NAME> --location <LOCATION> --project <PROJECT>"
  echo "  or: make uninstall-managed-csi CLUSTER_PROJECT=<PROJECT> CLUSTER_NAME=<CLUSTER_NAME> CLUSTER_LOCATION=<LOCATION>"
  exit 1
fi

echo "--------------------------------------"
echo "Uninstalling GKE Managed CSI Driver on the cluster"
echo "Target Project:  ${CLUSTER_PROJECT}"
echo "Target Cluster:  ${CLUSTER_NAME}"
echo "Target Location: ${CLUSTER_LOCATION}"
echo "--------------------------------------"

# Prompt for confirmation if running in an interactive terminal
if [[ -t 0 ]]; then
  read -r -p "Are you sure? (y/N): " confirm
  case "$(echo "${confirm}" | tr '[:upper:]' '[:lower:]')" in
    y|yes)
      ;;
    *)
      echo "Operation cancelled."
      exit 0
      ;;
  esac
fi

echo "Getting cluster credentials..."
gcloud container clusters get-credentials "${CLUSTER_NAME}" --location "${CLUSTER_LOCATION}" --project "${CLUSTER_PROJECT}"

echo "Checking CSI driver status on cluster..."
MANAGED_ADDON=$(gcloud container clusters describe "${CLUSTER_NAME}" \
  --location "${CLUSTER_LOCATION}" \
  --project "${CLUSTER_PROJECT}" \
  --format="value(addonsConfig.gcsFuseCsiDriverConfig.enabled)" 2>/dev/null || true)

if [[ "${MANAGED_ADDON}" != "True" ]]; then
  echo "GKE Managed Cloud Storage FUSE CSI driver is not installed on cluster '${CLUSTER_NAME}'. Nothing to uninstall."
  exit 0
fi

echo "Uninstalling GKE managed Cloud Storage FUSE CSI driver..."
gcloud container clusters update "${CLUSTER_NAME}" \
  --location="${CLUSTER_LOCATION}" \
  --project="${CLUSTER_PROJECT}" \
  --update-addons=GcsFuseCsiDriver=DISABLED

echo "GKE Managed Cloud Storage FUSE CSI driver successfully uninstalled."
