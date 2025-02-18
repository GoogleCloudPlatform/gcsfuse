#!/bin/bash
# Copyright 2024 Google LLC
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

# Fail on any error
set -eo pipefail

# Display commands being run
set -x

uname=$(uname -m)
if [ $uname == "aarch64" ];then
  # TODO: Remove this when we have an ARM64 image for the storage test bench.(b/384388821)
  echo "These tests will not run for arm64 machine..."
  exit 0
fi

RUN_E2E_TESTS_ON_PACKAGE=$1
# Only run on Go 1.17+
min_minor_ver=17

v=`go version | { read _ _ v _; echo ${v#go}; }`
comps=(${v//./ })
minor_ver=${comps[1]}

if [ "$minor_ver" -lt "$min_minor_ver" ]; then
    echo minor version $minor_ver, skipping
    exit 0
fi

# Install dependencies
# Ubuntu/Debian based machine.
if [ -f /etc/debian_version ]; then
  if grep -q "Ubuntu" /etc/os-release; then
    os="ubuntu"
  elif grep -q "Debian" /etc/os-release; then
    os="debian"
  fi

  sudo apt-get update
  sudo apt-get install -y ca-certificates curl
  sudo install -m 0755 -d /etc/apt/keyrings
  sudo curl -fsSL https://download.docker.com/linux/${os}/gpg -o /etc/apt/keyrings/docker.asc
  sudo chmod a+r /etc/apt/keyrings/docker.asc
  # Add the repository to Apt sources:
  echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/${os} \
    $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
    sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
  sudo apt-get update
  sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  sudo apt-get install -y lsof
# RHEL/CentOS based machine.
elif [ -f /etc/redhat-release ]; then
    sudo dnf -y install dnf-plugins-core
    sudo dnf config-manager --add-repo https://download.docker.com/linux/rhel/docker-ce.repo
    sudo dnf -y install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    sudo usermod -aG docker $USER
    sudo systemctl start docker
    sudo yum -y install lsof
fi

export STORAGE_EMULATOR_HOST="http://localhost:9000"

DEFAULT_IMAGE_NAME='gcr.io/cloud-devrel-public-resources/storage-testbench'
DEFAULT_IMAGE_TAG='latest'
DOCKER_IMAGE=${DEFAULT_IMAGE_NAME}:${DEFAULT_IMAGE_TAG}
CONTAINER_NAME=storage_testbench

# Note: --net=host makes the container bind directly to the Docker host’s network,
# with no network isolation. If we were to use port-mapping instead, reset connection errors
# would be captured differently and cause unexpected test behaviour.
# The host networking driver works only on Linux hosts.
# See more about using host networking: https://docs.docker.com/network/host/
DOCKER_NETWORK="--net=host"

# Get the docker image for the testbench
sudo docker pull $DOCKER_IMAGE

# Remove the docker container if it's already running.
CONTAINER_ID=$(sudo docker ps -aqf "name=$CONTAINER_NAME")
if [[ -n "$CONTAINER_ID" ]]; then
  echo "Container with ID:[$CONTAINER_ID] is already running with name:[$CONTAINER_NAME]"
  echo "Stoping container...."
  sudo docker stop $CONTAINER_ID
fi

# Start the testbench
sudo docker run --name $CONTAINER_NAME --rm -d $DOCKER_NETWORK $DOCKER_IMAGE
echo "Running the Cloud Storage testbench: $STORAGE_EMULATOR_HOST"
sleep 5

# Stop the testbench & cleanup environment variables
function cleanup() {
    echo "Cleanup testbench"
    sudo docker stop $CONTAINER_NAME
    unset STORAGE_EMULATOR_HOST;
}
trap cleanup EXIT

# Create the JSON file to create bucket
cat << EOF > test.json
{"name":"test-bucket"}
EOF

# Execute the curl command to create bucket on storagetestbench server.
curl -X POST --data-binary @test.json \
    -H "Content-Type: application/json" \
    "$STORAGE_EMULATOR_HOST/storage/v1/b?project=test-project"
rm test.json

# Run Write Stall Tests.
go test ./tools/integration_tests/emulator_tests/write_stall/... --integrationTest -v --testbucket=test-bucket -timeout 10m --testInstalledPackage=$RUN_E2E_TESTS_ON_PACKAGE

# Run Streaming Writes Failure Tests.
go test ./tools/integration_tests/emulator_tests/streaming_writes_failure/... -p 1 -short --integrationTest -v --testbucket=test-bucket --testOnCustomEndpoint=http://localhost:8020 -timeout 10m --testInstalledPackage=$RUN_E2E_TESTS_ON_PACKAGE
