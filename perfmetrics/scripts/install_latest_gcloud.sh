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
set -x

if [[ $# -ne 0 ]]; then
    echo "This script requires no argument."
    echo "Usage: $0"
    exit 1
fi

INSTALL_DIR="/usr/local" # Installation directory

install_latest_gcloud() {
    # Upgrade Python first, as gcloud requires a version between 3.9 and 3.13.
    # The upgrade_python3.sh script installs Python 3.11.9.
    "$(dirname "$0")/upgrade_python3.sh"
    # Set CLOUDSDK_PYTHON to point to the newly installed Python executable.
    export CLOUDSDK_PYTHON="$HOME/.local/python-3.11.9/bin/python3.11"

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
    # If this script is run in background or different shell then
    # export PATH needs to be called from the shell or use absolute gcloud path
    # or permanently add this path to path variable in bashrc.
    echo 'export PATH="/usr/local/google-cloud-sdk/bin:$PATH"' >> "$HOME/.bashrc"
    echo 'export CLOUDSDK_PYTHON="$HOME/.local/python-3.11.9/bin/python3.11"' >> "$HOME/.bashrc"
    set +u
    source "$HOME/.bashrc"
    set -u
    echo "gcloud Version is:"
    gcloud version
    echo "Gcloud is present at: $( (which gcloud) )"
    rm -f "$INSTALLATION_LOG"
fi
