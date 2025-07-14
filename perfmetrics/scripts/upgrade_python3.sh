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
INSTALL_PREFIX="/usr/local"

# Install dependencies silently
sudo apt update > /dev/null
sudo apt install -y \
  build-essential zlib1g-dev libncurses5-dev libgdbm-dev libnss3-dev \
  libssl-dev libreadline-dev libffi-dev curl libsqlite3-dev \
  libbz2-dev liblzma-dev tk-dev uuid-dev wget > /dev/null

# Download Python source silently
cd /usr/src
sudo wget -q https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tgz
sudo tar -xf Python-${PYTHON_VERSION}.tgz > /dev/null
cd Python-${PYTHON_VERSION}

# Configure silently
sudo ./configure --enable-optimizations --prefix=$INSTALL_PREFIX > /dev/null

# Build silently
sudo make -j"$(nproc)" > /dev/null

# Install silently
sudo make altinstall > /dev/null

# Install pip silently
sudo $INSTALL_PREFIX/bin/python3.11 -m ensurepip > /dev/null
sudo $INSTALL_PREFIX/bin/python3.11 -m pip install --upgrade pip > /dev/null

# Remove old alternatives silently
sudo update-alternatives --remove-all python3 > /dev/null 2>&1 || true
sudo update-alternatives --remove-all pip3 > /dev/null 2>&1 || true

# Register Python 3.11 as alternative silently
sudo update-alternatives --install /usr/bin/python3 python3 $INSTALL_PREFIX/bin/python3.11 1 > /dev/null
sudo update-alternatives --install /usr/bin/pip3 pip3 $INSTALL_PREFIX/bin/pip3.11 1 > /dev/null

# Set Python 3.11 as default silently
sudo update-alternatives --set python3 $INSTALL_PREFIX/bin/python3.11 > /dev/null
sudo update-alternatives --set pip3 $INSTALL_PREFIX/bin/pip3.11 > /dev/null

# Final version check (visible)
python3 --version
pip3 --version
