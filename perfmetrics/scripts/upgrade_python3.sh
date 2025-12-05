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
#!/bin/bash

set -e

PYTHON_VERSION=3.11.9
INSTALL_PREFIX="$HOME/.local/python-$PYTHON_VERSION"

if command -v apt-get &> /dev/null; then
    # For Debian/Ubuntu-based systems
    echo "Installing dependencies for building Python for Debian..."
    sudo apt-get update -y > /dev/null
    sudo apt-get install -y \
      build-essential zlib1g-dev libncurses5-dev libgdbm-dev libnss3-dev \
      libssl-dev libreadline-dev libffi-dev curl libsqlite3-dev \
      libbz2-dev liblzma-dev tk-dev uuid-dev wget > /dev/null
elif command -v yum &> /dev/null; then
    # For RHEL/CentOS-based systems
    echo "Installing dependencies for building Python on RHEL..."
    # For RHEL-based systems, use 'yum' to install packages.
    # The "Development Tools" group is equivalent to 'build-essential' on Debian.
    # The '-devel' packages provide the necessary header files for compilation.
    sudo yum -y groupinstall "Development Tools" > /dev/null
    sudo yum -y install \
          zlib-devel ncurses-devel gdbm-devel nss-devel openssl-devel \
          readline-devel libffi-devel curl sqlite-devel bzip2-devel \
          xz-devel tk-devel libuuid-devel wget > /dev/null
else
    exit 1
fi


# Download and build Python locally
cd /tmp
wget -q https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tgz
tar -xf Python-${PYTHON_VERSION}.tgz
cd Python-${PYTHON_VERSION}

echo "Configuring Python build for local install..."
./configure --enable-optimizations --prefix="$INSTALL_PREFIX" > /dev/null

echo "Building Python $PYTHON_VERSION..."
make -j"$(nproc)" > /dev/null

echo "Installing Python $PYTHON_VERSION locally at $INSTALL_PREFIX..."
make altinstall > /dev/null

echo "Python $PYTHON_VERSION installed at $INSTALL_PREFIX/bin/python3.11"
"$INSTALL_PREFIX/bin/python3.11" --version
