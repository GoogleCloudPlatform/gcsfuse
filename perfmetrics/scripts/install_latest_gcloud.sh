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

# install_latest_gcloud installs the latest version of the Google Cloud SDK.
#
# The function performs the following steps:
# 1. Upgrades Python to a specific version using a helper script.
# 2. Sets the CLOUDSDK_PYTHON environment variable for the installation.
# 3. Downloads and installs the Google Cloud SDK to a system-wide location.
# 4. Installs the 'alpha' component.
# 5. Persists the CLOUDSDK_PYTHON and PATH configuration to the user's .bashrc.
#
# This function requires sudo privileges for installation and for modifying .bashrc.
install_latest_gcloud() {
    local -r python_version="3.11.9"
    local -r python_bin_dir="/user/.local/python-$python_version/bin"
    local -r python_executable="$python_bin_dir/python3.11"
    local -r bashrc="$HOME/.bashrc"
    local -r gcloud_install_dir="${INSTALL_DIR}/google-cloud-sdk"

    # Ensure we have sudo privileges upfront if not running as root.
    if [[ $EUID -ne 0 ]]; then
        echo "Sudo privileges are required. Please enter your password if prompted."
        sudo -v
    fi

    # Upgrade Python environment.
    echo "Upgrading Python to version $python_version..."
    "$(dirname "$0")/upgrade_python3.sh"
    export CLOUDSDK_PYTHON="$python_executable"

    # Create a temporary directory for the installation files.
    # Use a trap to ensure the directory is cleaned up on exit, error, or interrupt.
    local temp_dir
    temp_dir=$(mktemp -d /tmp/gcloud_install_src.XXXXXX)
    trap 'echo "Cleaning up temporary directory..."; rm -rf "$temp_dir"' EXIT

    # Download and install gcloud.
    echo "Downloading and installing the latest Google Cloud SDK..."
    pushd "$temp_dir" > /dev/null
    wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
    
    echo "Removing existing gcloud installation at ${gcloud_install_dir}..."
    sudo rm -rf "${gcloud_install_dir}"
    
    echo "Extracting new gcloud version to ${INSTALL_DIR}..."
    sudo tar -C "$INSTALL_DIR" -xzf gcloud.tar.gz
    
    echo "Running gcloud install script..."
    sudo env CLOUDSDK_PYTHON="$CLOUDSDK_PYTHON" "${gcloud_install_dir}/install.sh" -q

    echo "Updating gcloud components..."
    sudo env CLOUDSDK_PYTHON="$CLOUDSDK_PYTHON" "${gcloud_install_dir}/bin/gcloud" components update -q
    
    echo "Installing alpha components..."
    sudo env CLOUDSDK_PYTHON="$CLOUDSDK_PYTHON" "${gcloud_install_dir}/bin/gcloud" components install alpha -q
    
    popd > /dev/null

    # --- Persistence ---
    echo "Persisting environment variables to $bashrc..."

    local -r cloudsdk_export='export CLOUDSDK_PYTHON="'"$python_executable"'"'
    if ! grep -qxF -- "$cloudsdk_export" "$bashrc"; then
        echo "$cloudsdk_export" >> "$bashrc"
        echo "   -> Added CLOUDSDK_PYTHON to $bashrc."
    else
        echo "   -> CLOUDSDK_PYTHON already configured in $bashrc."
    fi

    local -r path_export='export PATH="'"$python_bin_dir"':$PATH"'
    if ! grep -qxF -- "$path_export" "$bashrc"; then
        echo "$path_export" >> "$bashrc"
        echo "   -> Added Python bin directory to PATH in $bashrc."
    else
        echo "   -> Python bin directory already in PATH in $bashrc."
    fi

    echo "---"
    echo "Cloud SDK installation complete."
    echo "Please run 'source $bashrc' or start a new shell to apply the changes."
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
