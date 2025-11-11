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

install_latest_gcloud() {
    # Define the custom Python path for clarity
    local PYTHON_DIR="$HOME/.local/python-3.11.9"
    local PYTHON_EXEC="${PYTHON_DIR}/bin/python3.11"
    
    # Define the site-packages directory based on your custom Python install
    # NOTE: The exact path may vary (e.g., python3.11/site-packages)
    local PYTHON_SITE_PACKAGES="${PYTHON_DIR}/lib/python3.11/site-packages"

    # Upgrade Python first
    "$(dirname "$0")/upgrade_python3.sh"
    
    # --- CRITICAL ADDITION ---
    # Set PYTHONPATH to ensure 'gcloud' components can find their dependencies
    export PYTHONPATH="$PYTHON_SITE_PACKAGES"

    # Set CLOUDSDK_PYTHON to point to the newly installed Python executable.
    export CLOUDSDK_PYTHON="$PYTHON_EXEC"
    echo "cloud SDK python: " $CLOUDSDK_PYTHON

    local temp_dir
    temp_dir=$(mktemp -d /tmp/gcloud_install_src.XXXXXX)
    pushd "$temp_dir"
    
    wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
    sudo rm -rf "${INSTALL_DIR}/google-cloud-sdk"
    sudo tar -C "$INSTALL_DIR" -xzf gcloud.tar.gz
    
    # Use `sudo env` to pass BOTH CLOUDSDK_PYTHON AND PYTHONPATH variables
    sudo env CLOUDSDK_PYTHON="$CLOUDSDK_PYTHON" \
        PYTHONPATH="$PYTHONPATH" \
        "${INSTALL_DIR}/google-cloud-sdk/install.sh" -q
        
    sudo env CLOUDSDK_PYTHON="$CLOUDSDK_PYTHON" \
        PYTHONPATH="$PYTHONPATH" \
        "${INSTALL_DIR}/google-cloud-sdk/bin/gcloud" components update -q
        
    sudo env CLOUDSDK_PYTHON="$CLOUDSDK_PYTHON" \
        PYTHONPATH="$PYTHONPATH" \
        "${INSTALL_DIR}/google-cloud-sdk/bin/gcloud" components install alpha -q
        
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
    # If this script is run in background or different shell then
    # export PATH needs to be called from the shell or use absolute gcloud path
    # or permanently add this path to path variable in bashrc.
    export PATH="${INSTALL_DIR}/google-cloud-sdk/bin:$PATH"
    echo "gcloud Version is:"
    gcloud version
    echo "Gcloud is present at: $( (which gcloud) )"
    rm -f "$INSTALLATION_LOG"
fi
