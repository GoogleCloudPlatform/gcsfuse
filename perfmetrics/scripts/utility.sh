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

if [[ -v "$UTILITY_SH_SOURCED" ]]; then
    # The file has already been sourced, so return.
    return 0
else
    UTILITY_SH_SOURCED=true
fi

# Helpers for logging
log_info() {
    echo "[INFO] $(date +"%Y-%m-%d %H:%M:%S"): $1"
}

log_error() {
    echo "[ERROR] $(date +"%Y-%m-%d %H:%M:%S"): $1"
}

log_and_exit() {
    log_error "$1"
    exit 1
}

# Helpers for build tools.
get_build_tools() {
    log_info "Getting build tools..."
    if command -v apt-get &> /dev/null; then
        log_info "Installing build tools for Debian/Ubuntu-based distributions..."
        sudo apt-get update -y > /dev/null
        sudo apt-get install -y build-essential \
            zlib1g-dev \
            libncurses5-dev \
            libgdbm-dev \
            libnss3-dev \
            libssl-dev \
            libreadline-dev \
            libffi-dev \
            curl \
            libsqlite3-dev \
            libbz2-dev \
            liblzma-dev \
            tk-dev \
            uuid-dev \
            wget > /dev/null
    elif command -v yum &> /dev/null; then
        log_info "Installing build tools for RHEL/CentOS-based distributions..."
        # The "Development Tools" group is equivalent to 'build-essential' on Debian.
        # The '-devel' packages provide the necessary header files for compilation.
        sudo yum -y groupinstall "Development Tools" > /dev/null
        sudo yum -y install zlib-devel \
            ncurses-devel \
            nss-devel \
            openssl-devel \
            readline-devel \
            libffi-devel \
            curl \
            sqlite-devel \
            bzip2-devel \
            xz-devel \
            tk-devel \
            libuuid-devel \
            wget > /dev/null
    else 
        log_error "Unsupported distribution."
        exit 1
    fi
}

_get_python() {
    (
    # Exit on error, treat unset variables as errors, and propagate pipeline errors.
    set -euo pipefail

    local version="$1"
    local prefix="$2"
    local dir="$3"
    pushd "$dir" || log_and_exit "Failed to move to temporary build directory."
    wget -q "https://www.python.org/ftp/python/${version}/Python-${version}.tgz"
    tar -xf "Python-${version}.tgz"
    pushd "Python-${version}" || log_and_exit "Failed to move to extracted Python directory."
    sudo rm -rf "$prefix" # Remove existing python installation
    log_info "Installing python version ${version} at installation path ${prefix}/bin"
    ./configure --enable-optimizations --prefix="$prefix"
    make -j"$(nproc)"
    make altinstall
    popd || log_and_exit "Failed to move back to parent directory."
    popd || log_and_exit "Failed to move back to parent directory."
    )
}

get_python() {
    if [[ $# -ne 1 ]]; then
        echo "This method requires exactly one argument."
        echo "Usage: get_python <python-version>"
        echo "Example: get_python 3.11.9"
        exit 1
    fi
    get_build_tools
    local PYTHON_VERSION="$1"
    local INSTALLATION_DIR="/usr/local/python/${PYTHON_VERSION}"
    local log_file build_dir
    log_file=$(mktemp "/tmp/python_install_log_file.XXXXXX") || { log_and_exit "Unable to create python install log file"; }
    build_dir=$(mktemp -d "/tmp/python-build-dir.XXXXXX") || { log_and_exit "Unable to create temporary build directory"; }
    log_info "Installing python version ${PYTHON_VERSION} at installation path ${INSTALLATION_DIR}"
    if ! _get_python "$1" "$INSTALLATION_DIR" "$build_dir" >"$log_file" 2>&1; then
        log_error "Unable to install python version ${PYTHON_VERSION} at installation path ${INSTALLATION_DIR}/bin"
        cat "$log_file"
    else
        log_info "Successfully installed python version ${PYTHON_VERSION} at installation path ${INSTALLATION_DIR}/bin"
        export PATH="${INSTALLATION_DIR}/bin:$PATH"
        local major_minor
        major_minor=$(echo "$PYTHON_VERSION" | cut -d'.' -f1,2)
        log_info "Checking where python${major_minor} is installed"
        whereis "python${major_minor}"
        log_info "Checking installed python version with command $ python${major_minor} --version"
        "python${major_minor}" --version
        rm -rf "$build_dir"
    fi
}

_get_latest_gcloud() {
    (
    # Exit on error, treat unset variables as errors, and propagate pipeline errors.
    set -euo pipefail
    local python_path="$1"
    local install_dir="$2"
    local build_dir="$3"
    pushd "$build_dir" || log_and_exit "Failed to move to temporary build directory."
    wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
    sudo rm -rf "${install_dir}/google-cloud-sdk" # Remove existing gcloud installation
    sudo tar -C "$install_dir" -xzf gcloud.tar.gz
    export CLOUDSDK_PYTHON="$python_path" # For installation within this subshell
    "${install_dir}/google-cloud-sdk/install.sh" -q
    "${install_dir}/google-cloud-sdk/bin/gcloud" components update -q
    "${install_dir}/google-cloud-sdk/bin/gcloud" components install alpha -q
    popd
    )
}

get_latest_gcloud() {
    if [[ $# -ne 1 ]]; then
        echo "This method requires exactly one argument."
        echo "Usage: get_latest_gcloud <python-version>"
        echo "Example: get_latest_gcloud 3.11.9"
        exit 1
    fi
    local PYTHON_VERSION="$1"
    local INSTALLATION_DIR="/usr/local"
    local python_path log_file build_dir
    log_file=$(mktemp "/tmp/gcloud_install_log_file.XXXXXX") || { log_and_exit "Unable to create gcloud install log file"; }
    build_dir=$(mktemp -d "/tmp/gcloud-build-dir.XXXXXX") || { log_and_exit "Unable to create temporary build directory"; }
    get_python "$1"
    local major_minor
    major_minor=$(echo "$PYTHON_VERSION" | cut -d'.' -f1,2)
    python_path=$(which "python${major_minor}")
    log_info "Installing latest gcloud version at installation path ${INSTALLATION_DIR}/google-cloud-sdk/bin"
    if ! _get_latest_gcloud "$python_path" "$INSTALLATION_DIR" "$build_dir" >"$log_file" 2>&1; then
        log_error "Unable to install latest gcloud version with python version ${PYTHON_VERSION} at installation path ${INSTALLATION_DIR}/google-cloud-sdk/bin"
        cat "$log_file"
    else
        log_info "Successfully installed latest gcloud at installation path ${INSTALLATION_DIR}/google-cloud-sdk/bin"
        export PATH="${INSTALLATION_DIR}/google-cloud-sdk/bin:$PATH"
        export CLOUDSDK_PYTHON="$python_path" # For callers of get_latest_gcloud
        local major_minor
        log_info "Checking where gcloud is installed"
        whereis gcloud
        log_info "Checking installed gcloud version with command $ python${major_minor} --version"
        gcloud --version
        rm -rf "$build_dir"
    fi
}