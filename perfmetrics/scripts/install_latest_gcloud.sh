#!/bin/bash
# Copyright 2025 Google LLC
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

# Script to install latest version of gcloud along with alpha components

# Exit on error, treat unset variables as errors, and propagate pipeline errors.
set -euo pipefail

if [[ $# -ne 0 ]]; then
    echo "This script requires no argument."
    echo "Usage: $0"
    exit 1
fi

INSTALL_DIR="/usr/local" # Installation directory

# Upgrade Python first, as gcloud requires a version between 3.9 and 3.13.
# The upgrade_python3.sh script installs Python 3.11.9 (or skips if already installed).
"$(dirname "$0")/upgrade_python3.sh"
export CLOUDSDK_PYTHON="$HOME/.local/python-3.11.9/bin/python3.11"
export PATH="$HOME/.local/python-3.11.9/bin:$PATH"

GCLOUD_BIN=""
CURRENT_VER=""

check_existing_gcloud() {
    if [[ -x "${INSTALL_DIR}/google-cloud-sdk/bin/gcloud" ]]; then
        GCLOUD_BIN="${INSTALL_DIR}/google-cloud-sdk/bin/gcloud"
    fi

    if [[ -n "$GCLOUD_BIN" ]]; then
        local gcloud_ver
        gcloud_ver=$("$GCLOUD_BIN" version 2>/dev/null || true)
        CURRENT_VER=$(echo "$gcloud_ver" | grep "Google Cloud SDK" | awk '{print $4}')
        local has_alpha=false
        if echo "$gcloud_ver" | grep -q "alpha"; then
            has_alpha=true
        fi

        # Fetch latest published version from Google Cloud SDK rapid release channel manifest
        local latest_ver
        latest_ver=$(curl -s --connect-timeout 5 https://dl.google.com/dl/cloudsdk/channels/rapid/components-2.json | grep -m 1 -o '"version": "[^"]*"' | cut -d'"' -f4 || true)

        # Ensure local version matches the latest available version and has alpha component
        if [[ -n "$latest_ver" && "$CURRENT_VER" == "$latest_ver" && "$has_alpha" == "true" ]]; then
            return 0
        fi
    fi
    return 1
}

if check_existing_gcloud; then
    export PATH="${INSTALL_DIR}/google-cloud-sdk/bin:$PATH"
    echo "gcloud is already at the latest version ($CURRENT_VER) with alpha component installed at $GCLOUD_BIN."
    exit 0
fi

install_latest_gcloud() {
    set -x

    local temp_dir
    temp_dir=$(mktemp -d /tmp/gcloud_install_src.XXXXXX)
    pushd "$temp_dir"
    
    wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
    sudo rm -rf "${INSTALL_DIR}/google-cloud-sdk" # Remove existing gcloud installation
    sudo tar -C "$INSTALL_DIR" -xzf gcloud.tar.gz
    # Use `sudo env` to pass the CLOUDSDK_PYTHON variable to the gcloud commands.
    sudo env CLOUDSDK_PYTHON="$CLOUDSDK_PYTHON" "${INSTALL_DIR}/google-cloud-sdk/install.sh" -q
    sudo env CLOUDSDK_PYTHON="$CLOUDSDK_PYTHON" "${INSTALL_DIR}/google-cloud-sdk/bin/gcloud" components update -q
    sudo env CLOUDSDK_PYTHON="$CLOUDSDK_PYTHON" "${INSTALL_DIR}/google-cloud-sdk/bin/gcloud" components install alpha -q
    popd
    sudo rm -rf "$temp_dir"
}

echo "Installing latest gcloud version to ${INSTALL_DIR}"
INSTALLATION_LOG=$(mktemp /tmp/gcloud_install_log.XXXXXX)
if ! install_latest_gcloud >"$INSTALLATION_LOG" 2>&1; then
    echo "latest gcloud installation failed."
    cat "$INSTALLATION_LOG"
    rm -f "$INSTALLATION_LOG"
    exit 1
else
    echo "latest gcloud installed successfully."
    echo "gcloud Version is:"
    export PATH="/usr/local/google-cloud-sdk/bin:$PATH"
    gcloud version
    echo "Gcloud is present at: $( (which gcloud) )"
    rm -f "$INSTALLATION_LOG"
fi
