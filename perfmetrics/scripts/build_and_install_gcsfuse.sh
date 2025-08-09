#!/bin/bash
# Copyright 2023 Google LLC
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

# This script will build gcsfuse package on given commitId or branch and install it on the machine.
# This will stop execution when any command will have non-zero status.
set -e

# --- Determine architecture (e.g., amd64, arm64) ---
architecture=$(dpkg --print-architecture)

# Install Docker if it's not already present or if this is a Kokoro environment as it has old docker version.
if ! command -v docker &> /dev/null || [[ -n "${KOKORO_ARTIFACTS_DIR}" ]]; then
    echo "Installing Docker..."
    sudo mkdir -p /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    echo \
      "deb [arch=${architecture} signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
      $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    sudo apt-get update
    sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin -y
else
    echo "Docker is already installed. Skipping Docker installation."
fi

# --- Build and install gcsfuse ---
echo "Building and installing gcsfuse..."
branch=$1
if [ -z "$branch" ]; then
  echo "Usage: $0 <branch-or-commit-id>"
  exit 1
fi

GCSFUSE_VERSION=0.0.0

# Build the gcsfuse package using Docker
sudo docker buildx build --load ./tools/package_gcsfuse_docker/ -t gcsfuse:$branch --build-arg ARCHITECTURE=${architecture} --build-arg GCSFUSE_VERSION=$GCSFUSE_VERSION --build-arg BRANCH_NAME=$branch --platform=linux/${architecture}
sudo docker run -v $HOME/release:/release gcsfuse:$branch cp -r /packages /release/
sudo dpkg -i $HOME/release/packages/gcsfuse_${GCSFUSE_VERSION}_${architecture}.deb
