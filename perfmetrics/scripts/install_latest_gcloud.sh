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
    # Define the necessary paths
    local python_bin_dir="$INSTALL_DIR/.local/python-3.11.9/bin"
    local python_executable="$python_bin_dir/python3.11"
    local bashrc="$HOME/.bashrc" # Use "$HOME/.zshrc" for Zsh

    # --- Existing Python Upgrade and Variable Setup for *Current* Session ---
    "$(dirname "$0")/upgrade_python3.sh"
    # Set CLOUDSDK_PYTHON for the *current* script/shell execution
    export CLOUDSDK_PYTHON="$python_executable"

    # --- Cloud SDK Installation Logic ---
    local temp_dir
    temp_dir=$(mktemp -d /tmp/gcloud_install_src.XXXXXX)
    pushd "$temp_dir"

    # ... (rest of your installation logic) ...
    wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
    sudo rm -rf "${INSTALL_DIR}/google-cloud-sdk"
    sudo tar -C "$INSTALL_DIR" -xzf gcloud.tar.gz
    sudo env CLOUDSDK_PYTHON="$CLOUDSDK_PYTHON" "${INSTALL_DIR}/google-cloud-sdk/install.sh" -q
    sudo env CLOUDSDK_PYTHON="$CLOUDSDK_PYTHON" "${INSTALL_DIR}/google-cloud-sdk/bin/gcloud" components update -q
    sudo env CLOUDSDK_PYTHON="$CLOUDSDK_PYTHON" "${INSTALL_DIR}/google-cloud-sdk/bin/gcloud" components install alpha -q
    popd
    sudo rm -rf "$temp_dir"

    # ---------------------------------------
    # --- CORRECTED PERSISTENCE STEP ---
    # ---------------------------------------
    
    # 1. Persist CLOUDSDK_PYTHON
    echo ""
    echo "✅ Persisting CLOUDSDK_PYTHON to $bashrc..."
    local export_cloudsdk='export CLOUDSDK_PYTHON="'"$python_executable"'"'
    
    # Check if the line already exists to avoid duplication
    if ! grep -qxF -- "$export_cloudsdk" "$bashrc"; then
        echo "$export_cloudsdk" >> "$bashrc"
        echo "   -> Added CLOUDSDK_PYTHON export."
    fi

    # 2. Persist PATH addition
    echo "✅ Persisting PATH addition ($python_bin_dir) to $bashrc..."
    local export_path='export PATH="'"$python_bin_dir"':$PATH"'

    # Check if the line already exists to avoid duplication
    if ! grep -qxF -- "$export_path" "$bashrc"; then
        echo "$export_path" >> "$bashrc"
        echo "   -> Added Python bin directory to PATH export."
    fi

    # 3. Source the file to apply the change immediately to the parent shell
    source "$bashrc"

    echo "---"
    echo "Cloud SDK installation complete. Variables are now set for all new shells."
    echo "CLOUDSDK_PYTHON set to: $CLOUDSDK_PYTHON"
    echo "Current PATH includes: $python_bin_dir"
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
