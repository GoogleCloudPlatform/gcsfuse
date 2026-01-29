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

# Script to install a specific go version to /usr/local
# Usage: install_go.sh <go-version>

# Exit on error, treat unset variables as errors, and propagate pipeline errors.
set -euo pipefail

if [[ $# -ne 1 ]]; then
    echo "This script requires exactly one argument."
    echo "Usage: $0 <go-version>"
    echo "Example: $0 1.24.0"
    exit 1
fi

# Source common utilities for OS and Arch detection
SCRIPT_DIR=$(dirname "$(readlink -f "$0")")
if [ -f "${SCRIPT_DIR}/os_utils.sh" ]; then
    source "${SCRIPT_DIR}/os_utils.sh"
else
    echo "Error: os_utils.sh not found in ${SCRIPT_DIR}"
    exit 1
fi

GO_VERSION="$1"
INSTALL_DIR="/usr/local" # Installation directory

# Function to download, extract, and install go
install_go() {
    local temp_dir architecture os_id system_arch
    
    if ! os_id=$(get_os_id); then
      log_error "Failed to detect OS ID."
      return 1
    fi
    architecture=$(get_go_arch)
    system_arch=$(uname -m)

    if [ "$architecture" == "unsupported" ]; then
      echo "Unsupported architecture: $system_arch"
      return 1
    fi
    echo "Detected architecture: $system_arch mapped to Go arch: $architecture"

    echo "Installing dependencies for Go installation..."
    install_packages_by_os "$os_id" "wget" "tar" || {
      echo "Warning: Could not install dependencies via package manager. Proceeding with existing tools."
    }

    temp_dir=$(mktemp -d /tmp/go_install_src.XXXXXX)
    pushd "$temp_dir"

    local url="https://go.dev/dl/go${GO_VERSION}.linux-${architecture}.tar.gz"
    echo "Downloading from: $url"
    wget -O go_tar.tar.gz "https://go.dev/dl/go${GO_VERSION}.linux-${architecture}.tar.gz" -q

    sudo rm -rf "${INSTALL_DIR}/go" # Remove previous installation.
    sudo tar -C "$INSTALL_DIR" -xzf go_tar.tar.gz
    
    popd
    sudo rm -rf "$temp_dir"
}

echo "Installing Go version ${GO_VERSION} to ${INSTALL_DIR}"
INSTALLATION_LOG=$(mktemp /tmp/go_install_log.XXXXXX)
if ! install_go > "$INSTALLATION_LOG" 2>&1; then
    echo "Go version ${GO_VERSION} installation failed."
    echo "--- Installation Log ---"
    cat "$INSTALLATION_LOG"
    echo "------------------------"
    rm -f "$INSTALLATION_LOG"
    exit 1
else
    echo "Go version ${GO_VERSION} installed successfully."
    # If this script is run in background or different shell then
    # export PATH needs to be called from the shell or use absolute go path
    # or permanently add this to path variable in bashrc.
    export PATH="${INSTALL_DIR}/go/bin:$PATH"
    
    # Verify installation
    if ! command -v go >/dev/null 2>&1; then
      echo "Error: 'go' command not found after installation. Check path."
      cat "$INSTALLATION_LOG"
      exit 1
    fi
    
    echo "Go version is: $(go version)"
    echo "Go is present at: $(which go)"
    rm -f "$INSTALLATION_LOG"
fi
