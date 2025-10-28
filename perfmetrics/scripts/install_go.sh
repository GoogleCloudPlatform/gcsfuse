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

# Logging Helpers
log_info() {
    echo "[$(date +"%H:%M:%S %Z")] INFO: $1"
}

log_error() {
    echo "[$(date +"%H:%M:%S %Z")] ERROR: $1"
}

if [[ $# -ne 1 ]]; then
    log_error "This script requires exactly one argument."
    log_info "Usage: $0 <go-version>"
    log_info "Example: $0 1.24.5"
    exit 1
fi

GO_VERSION="$1"
INSTALL_DIR="/usr/local" # Installation directory

# Function to download, extract, and install go
install_go() {
    local temp_dir architecture
    temp_dir=$(mktemp -d /tmp/go_install_src.XXXXXX)
    pushd "$temp_dir"

    architecture=$(dpkg --print-architecture)
    wget -O go_tar.tar.gz "https://go.dev/dl/go${GO_VERSION}.linux-${architecture}.tar.gz" -q
    sudo rm -rf "${INSTALL_DIR}/go" # Remove previous installation.
    sudo tar -C "$INSTALL_DIR" -xzf go_tar.tar.gz
    
    popd
    sudo rm -rf "$temp_dir"
}

log_info "Installing Go version ${GO_VERSION} to ${INSTALL_DIR}"
INSTALLATION_LOG=$(mktemp /tmp/go_install_log.XXXXXX)
if ! install_go > "$INSTALLATION_LOG" 2>&1; then
    log_error "Go version ${GO_VERSION} installation failed."
    cat "$INSTALLATION_LOG"
    rm -f "$INSTALLATION_LOG"
    exit 1
else
    log_info "Go version ${GO_VERSION} installed successfully."
    # If this script is run in background or different shell then
    # export PATH needs to be called from the shell or use absolute go path
    # or permanently add this to path variable in bashrc.
    export PATH="${INSTALL_DIR}/go/bin:$PATH"
    log_info "Go version is: "
    go version
    log_info "Go is present at: $( (which go) )"
    rm -f "$INSTALLATION_LOG"
fi
