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
    echo "Example: $0 1.24.11"
    exit 1
fi

GO_VERSION="$1"
INSTALL_DIR="/usr/local" # Installation directory

# Function to download, extract, and install go
install_go() {
    local temp_dir architecture system_arch
    temp_dir=$(mktemp -d /tmp/go_install_src.XXXXXX)
    pushd "$temp_dir" > /dev/null

    # Detect Architecture regardless of Distro (RHEL/Arch/Debian)
    # uname -m returns x86_64 on Intel/AMD and aarch64 on ARM
    system_arch=$(uname -m)
    case "$system_arch" in
        x86_64)
            architecture="amd64"
            ;;
        aarch64|arm64)
            architecture="arm64"
            ;;
        *)
            echo "Unsupported architecture: $system_arch" >&2
            exit 1
            ;;
    esac

    echo "Detected architecture: $system_arch mapped to Go arch: $architecture"

    # Download with retries
    local url="https://go.dev/dl/go${GO_VERSION}.linux-${architecture}.tar.gz"
    echo "Downloading from: $url"
    
    # Check if wget exists, otherwise try curl
    if command -v wget >/dev/null 2>&1; then
        wget -O go_tar.tar.gz "$url" -q
    elif command -v curl >/dev/null 2>&1; then
        curl -s -L -o go_tar.tar.gz "$url"
    else
        echo "Error: Neither wget nor curl is installed." >&2
        exit 1
    fi

    sudo rm -rf "${INSTALL_DIR}/go" # Remove previous installation.
    sudo tar -C "$INSTALL_DIR" -xzf go_tar.tar.gz
    
    popd > /dev/null
    sudo rm -rf "$temp_dir"
}

echo "Installing Go version ${GO_VERSION} to ${INSTALL_DIR}"
INSTALLATION_LOG=$(mktemp /tmp/go_install_log.XXXXXX)
# We redirect stdout/stderr to log, but if it fails, we cat the log.
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
