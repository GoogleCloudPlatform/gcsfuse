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

if ! install_bash >"$INSTALLATION_LOG" 2>&1; then
    echo "Bash version ${BASH_VERSION} installation failed."
    cat "$INSTALLATION_LOG"
    rm -f "$INSTALLATION_LOG"
    exit 1
else
    echo "Bash ${BASH_VERSION} installed successfully."
    echo "Checking bash version at ${INSTALL_DIR}bin/bash:"
    "${INSTALL_DIR}bin/bash" --version
    rm -f "$INSTALLATION_LOG"
fi
