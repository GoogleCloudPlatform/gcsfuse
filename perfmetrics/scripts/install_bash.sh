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

# Script to install a specific version of GNU Bash to /usr/local/bin/bash
# Usage: install_bash.go <bash-version>

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
    log_info "Usage: $0 <bash-version>"
    log_info "Example: $0 5.3"
    exit 1
fi

BASH_VERSION="$1"
INSTALL_DIR="/usr/local/" # Installation directory

# Function to install dependencies like gcc and make if not present
install_dependencies() {
    if ! command -v gcc &>/dev/null || ! command -v make &>/dev/null; then
        log_info "GCC or make not found. Attempting to install build tools..."
        if command -v apt-get &>/dev/null; then
            sudo apt-get update && sudo apt-get install -y build-essential
        elif command -v dnf &>/dev/null; then
            sudo dnf install -y gcc make
        elif command -v yum &>/dev/null; then
            sudo yum install -y gcc make
        else
            log_error "Error: Could not find a known package manager (apt, dnf, yum)."
            log_error "Please install gcc and make manually before running this script."
            exit 1
        fi
        log_info "Build tools installed successfully."
    fi
}

# Function to download, compile, and install Bash
install_bash() {
    (
    set -euo pipefail
    local temp_dir
    temp_dir=$(mktemp -d /tmp/bash_install_src.XXXXXX)
    pushd "$temp_dir"

    wget -q "https://ftp.gnu.org/gnu/bash/bash-${BASH_VERSION}.tar.gz"
    tar -xzf "bash-${BASH_VERSION}.tar.gz"
    cd "bash-${BASH_VERSION}"
    ./configure --prefix="$INSTALL_DIR" --enable-readline
    make -s -j"$(nproc 2>/dev/null || echo 1)"
    sudo make install

    popd
    rm -rf "$temp_dir"
    )
}

log_info "Installing bash version ${BASH_VERSION} to ${INSTALL_DIR}bin/bash"
INSTALLATION_LOG=$(mktemp /tmp/bash_install_log.XXXXXX)

# Installing dependencies before installing Bash
install_dependencies
set +e
install_bash >"$INSTALLATION_LOG" 2>&1
installation_status=$?
set -e
if [[ $installation_status -ne 0 ]]; then
    log_error "Bash version ${BASH_VERSION} installation failed."
    cat "$INSTALLATION_LOG"
    rm -f "$INSTALLATION_LOG"
    exit 1
else
    log_info "Bash ${BASH_VERSION} installed successfully."
    log_info "Checking bash version at ${INSTALL_DIR}bin/bash:"
    "${INSTALL_DIR}bin/bash" --version
    rm -f "$INSTALLATION_LOG"
fi
