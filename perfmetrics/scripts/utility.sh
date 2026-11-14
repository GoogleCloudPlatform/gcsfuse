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

if [ -n "${UTILITY_SH_SOURCED-}" ]; then
    # The file has been sourced already, so return.
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
install_build_tools() {
    log_info "Installing build tools..."
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

_install_python() {
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
    sudo make altinstall
    popd || log_and_exit "Failed to move back to parent directory."
    popd || log_and_exit "Failed to move back to parent directory."
    )
}

install_python() {
    if [[ $# -ne 1 ]]; then
        echo "This method requires exactly one argument."
        echo "Usage: install_python <python-version>"
        echo "Example: install_python 3.11.9"
        exit 1
    fi

    install_build_tools
    local PYTHON_VERSION="$1"
    local INSTALLATION_DIR="/usr/local/python/${PYTHON_VERSION}"
    local log_file build_dir
    log_file=$(mktemp "/tmp/python_install_log_file.XXXXXX") || { log_and_exit "Unable to create python install log file"; }
    build_dir=$(mktemp -d "/tmp/python-build-dir.XXXXXX") || { log_and_exit "Unable to create temporary build directory"; }
    log_info "Installing python version ${PYTHON_VERSION} at installation path ${INSTALLATION_DIR}"
    _install_python "$1" "$INSTALLATION_DIR" "$build_dir" >"$log_file" 2>&1 
    local success=$?
    if [[ $success -ne 0 ]]; then
        log_error "Unable to install python version ${PYTHON_VERSION} at installation path ${INSTALLATION_DIR}/bin"
        cat "$log_file"
    else
        log_info "Successfully installed python version ${PYTHON_VERSION} at installation path ${INSTALLATION_DIR}/bin"
        export PATH="${INSTALLATION_DIR}/bin:$PATH"
        local major_minor
        major_minor=$(echo "$PYTHON_VERSION" | cut -d'.' -f1,2)
        log_info "Checking from which location we are using python${major_minor}"
        which "python${major_minor}"
        log_info "Checking installed python version with command $ python${major_minor} --version"
        "python${major_minor}" --version
        sudo rm -rf "$build_dir"
    fi
}

_install_latest_gcloud() {
    (
    # Exit on error, treat unset variables as errors, and propagate pipeline errors.
    set -euo pipefail
    local install_dir="$1"
    local build_dir="$2"
    pushd "$build_dir" || log_and_exit "Failed to move to temporary build directory."
    wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
    sudo rm -rf "${install_dir}/google-cloud-sdk" # Remove existing gcloud installation
    sudo tar -C "$install_dir" -xzf gcloud.tar.gz
    sudo "${install_dir}/google-cloud-sdk/install.sh" -q
    sudo "${install_dir}/google-cloud-sdk/bin/gcloud" components update -q
    sudo "${install_dir}/google-cloud-sdk/bin/gcloud" components install alpha -q
    popd
    )
}

# Installing latest gcloud requires python version compatibility.
# See here for supported python version for latest gcloud https://docs.cloud.google.com/sdk/docs/install#linux
install_latest_gcloud() {
    if [[ $# -ne 1 ]]; then
        echo "This method requires exactly one argument."
        echo "Usage: install_latest_gcloud <python-version-for-gcloud>"
        echo "Example: install_latest_gcloud 3.11.9"
        exit 1
    fi
    local PYTHON_VERSION="$1"
    local INSTALLATION_DIR="/usr/local"
    local python_path log_file build_dir
    log_file=$(mktemp "/tmp/gcloud_install_log_file.XXXXXX") || { log_and_exit "Unable to create gcloud install log file"; }
    build_dir=$(mktemp -d "/tmp/gcloud-build-dir.XXXXXX") || { log_and_exit "Unable to create temporary build directory"; }
    install_python "$PYTHON_VERSION"
    local major_minor
    major_minor=$(echo "$PYTHON_VERSION" | cut -d'.' -f1,2)
    python_path=$(which "python${major_minor}")
    export CLOUDSDK_PYTHON="$python_path" # CLOUDSDK_PYTHON env is required for gcloud installation & commands to work.
    log_info "Installing latest gcloud version at installation path ${INSTALLATION_DIR}/google-cloud-sdk/bin"
    _install_latest_gcloud "$INSTALLATION_DIR" "$build_dir" >"$log_file" 2>&1
    local success=$?
    if [[ $success -ne 0 ]]; then
        log_error "Unable to install latest gcloud version with python version ${PYTHON_VERSION} at installation path ${INSTALLATION_DIR}/google-cloud-sdk/bin"
        cat "$log_file"
    else
        log_info "Successfully installed latest gcloud at installation path ${INSTALLATION_DIR}/google-cloud-sdk/bin"
        export PATH="${INSTALLATION_DIR}/google-cloud-sdk/bin:$PATH"
        log_info "Checking from which location we are using gcloud"
        which gcloud
        log_info "Checking installed gcloud version with command $ gcloud --version"
        gcloud --version
        sudo rm -rf "$build_dir"
    fi
}