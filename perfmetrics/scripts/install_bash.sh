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

if [[ $# -ne 1 ]]; then
    echo "This script requires exactly one argument."
    echo "Usage: $0 <bash-version>"
    echo "Example: $0 5.1"
    exit 1
fi

BASH_VERSION="$1"
INSTALL_DIR="/usr/local/" # Installation directory

# Function to install dependencies like gcc and make if not present
install_dependencies() {
    if ! command -v gcc &>/dev/null || ! command -v make &>/dev/null; then
        echo "GCC or make not found. Attempting to install build tools..."
        if command -v apt-get &>/dev/null; then
            sudo apt-get update && sudo apt-get install -y build-essential
        elif command -v dnf &>/dev/null; then
            sudo dnf install -y gcc make
        elif command -v yum &>/dev/null; then
            sudo yum install -y gcc make
        else
            echo "Error: Could not find a known package manager (apt, dnf, yum)."
            echo "Please install gcc and make manually before running this script."
            exit 1
        fi
        echo "Build tools installed successfully."
    fi
}

# Function to download, compile, and install Bash
install_bash() {
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
}

echo "Installing bash version ${BASH_VERSION} to ${INSTALL_DIR}bin/bash"
INSTALLATION_LOG=$(mktemp /tmp/bash_install_log.XXXXXX)

# Installing dependencies before installing Bash
install_dependencies

if ! install_bash >"$INSTALLATION_LOG" 2>&1; then
    echo "Error: Bash version ${BASH_VERSION} installation failed."
    cat "$INSTALLATION_LOG"
    rm -f "$INSTALLATION_LOG"
    exit 1
else
    echo "Bash ${BASH_VERSION} installation command finished."
    EXPECTED_BASH_PATH="${INSTALL_DIR}bin/bash"

    if [ -f "${EXPECTED_BASH_PATH}" ]; then
        echo "Bash binary was created at the expected path: ${EXPECTED_BASH_PATH}"
        if [[ ":$PATH:" == *":${INSTALL_DIR}bin:"* ]]; then
            echo "The directory ${INSTALL_DIR}bin is in the PATH."
        else
            echo "Warning: The directory ${INSTALL_DIR}bin is NOT in the PATH."
        fi
    else
        echo "Warning: Bash binary not found at the expected path: ${EXPECTED_BASH_PATH}"
    fi

    echo "Verifying 'bash' is accessible via the PATH..."
    # Clear the shell's command hash to ensure we find the new one if it was just installed to the path.
    hash -r
    if ! command -v bash &> /dev/null; then
        echo "Error: No 'bash' executable found in the PATH. Cannot proceed."
        rm -f "$INSTALLATION_LOG"
        exit 1
    else
        SYSTEM_BASH_PATH=$(which bash)
        echo "The 'bash' command resolves to: ${SYSTEM_BASH_PATH}"
        echo "Version of this bash is:"
        "${SYSTEM_BASH_PATH}" --version
    fi

    rm -f "$INSTALLATION_LOG"
fi
