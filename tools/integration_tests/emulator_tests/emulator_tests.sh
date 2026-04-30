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

# Exit on error, treat unset variables as errors, and propagate pipeline errors.
set -euo pipefail
set -x

# Script uses positional arguments:
#   $1: testInstalledPackage (default: false)
#   $2: gcsfuse prebuilt directory (default: "")

# Logging Helpers
log_info() {
  echo "[INFO] $(date +"%Y-%m-%d %H:%M:%S"): $1"
}

log_error() {
  echo "[ERROR] $(date +"%Y-%m-%d %H:%M:%S"): $1"
}



TEST_INSTALLED_PACKAGE=${1:-false}
GCSFUSE_PREBUILT_DIR=${2:-""}

if [[ "$TEST_INSTALLED_PACKAGE" == "true" ]] && [[ -n "$GCSFUSE_PREBUILT_DIR" ]]; then
  log_error "test_installed_package=true and gcsfuse_prebuilt_dir are mutually exclusive."
  exit 1
fi


uname=$(uname -m)
if [ $uname == "aarch64" ];then
  # TODO: Remove this when we have an ARM64 image for the storage test bench.(b/384388821)
  log_info "These tests will not run for arm64 machine..."
  exit 0
fi

# Only run on Go 1.17+
min_minor_ver=17

v=`go version | { read _ _ v _; echo ${v#go}; }`
comps=(${v//./ })
minor_ver=${comps[1]}

if [ "$minor_ver" -lt "$min_minor_ver" ]; then
    log_info "minor version $minor_ver, skipping"
    exit 0
fi

# Install dependencies
if sudo docker ps > /dev/null 2>&1; then
  log_info "Docker is already installed and usable. Skipping installation steps."
else
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
fi

export STORAGE_EMULATOR_HOST="http://localhost:9000"
export STORAGE_EMULATOR_HOST_GRPC="localhost:8888"

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
  log_info "Container with ID:[$CONTAINER_ID] is already running with name:[$CONTAINER_NAME]"
  log_info "Stopping and removing container...."
  docker stop $CONTAINER_ID || true
  docker rm $CONTAINER_ID || true
fi

wait_for_emulator() {
  local timeout=60
  local count=0
  log_info "Waiting for emulator to be ready..."
  while [ $count -lt $timeout ]; do
    if curl -s "$STORAGE_EMULATOR_HOST/storage/v1/b?project=test-project" > /dev/null; then
      log_info "Emulator is ready!"
      return 0
    fi
    sleep 1
    count=$((count+1))
  done
  log_error "Emulator failed to become ready after $timeout seconds."
  return 1
}

# Run the emulator container in the background and stream its logs to a file.
# GUNICORN_CMD_ARGS="--timeout 600" is used to increase the Gunicorn timeout from the default 30 seconds
# to 10 minutes, preventing workers from dying during high load which causes emulator test flakiness.
sudo docker run --name $CONTAINER_NAME -e GUNICORN_CMD_ARGS="--timeout 600" --rm -d $DOCKER_NETWORK $DOCKER_IMAGE
log_info "Emulator docker container logs are saved at: $(pwd)/emulator_container.log"
sudo docker logs -f $CONTAINER_NAME > emulator_container.log 2>&1 &

# Stop the testbench & cleanup environment variables
function cleanup() {
    log_info "Cleanup testbench"
    sudo docker stop $CONTAINER_NAME || true
    unset STORAGE_EMULATOR_HOST;
    unset STORAGE_EMULATOR_HOST_GRPC;
    log_info "Printing emulator docker container Logs..."
    log_info "========================================================================"
    cat emulator_container.log || true
    log_info "========================================================================"
    rm -f test.json || true
}
trap cleanup EXIT

wait_for_emulator

# Create the JSON file to create bucket
cat << EOF > test.json
{"name":"test-bucket"}
EOF

# Execute the curl command to create bucket on storagetestbench server.
if ! curl -X POST --data-binary @test.json \
    -H "Content-Type: application/json" \
    "$STORAGE_EMULATOR_HOST/storage/v1/b?project=test-project"; then
  log_error "Failed to create bucket test-bucket"
  exit 1
fi
rm test.json

# Create an HNS bucket for control client tests
cat << EOF > test_hns.json
{"name":"test-hns-bucket", "hierarchicalNamespace": {"enabled": true}}
EOF
if ! curl -X POST --data-binary @test_hns.json -H "Content-Type: application/json" "$STORAGE_EMULATOR_HOST/storage/v1/b?project=test-project"; then
  log_error "Failed to create bucket test-hns-bucket"
  exit 1
fi
rm test_hns.json


# Start the gRPC server on port 8888.
log_info "Starting the gRPC server on port 8888"
response=$(curl -w "%{http_code}\n" --retry 5 --retry-max-time 40 -o /dev/null "$STORAGE_EMULATOR_HOST/start_grpc?port=8888")

if [[ $response != 200 ]]
then
    log_error "Testbench gRPC server did not start correctly"
    exit 1
fi

args=("--testInstalledPackage=$TEST_INSTALLED_PACKAGE")

if [[ -n "$GCSFUSE_PREBUILT_DIR" ]]; then
  args+=("--gcsfuse_prebuilt_dir=$GCSFUSE_PREBUILT_DIR")
fi

# Run all emulator test packages in sequence to avoid high cpu usage.
TEST_TARGET=${TEST_TARGET:-"./tools/integration_tests/emulator_tests/..."}
# Run all emulator test packages in sequence to avoid high cpu usage.
# Run all other emulator tests with standard bucket
go test -v -p 1 -timeout 10m $(go list ${TEST_TARGET} | grep -v control_client_stall) --integrationTest --testbucket=${TEST_BUCKET:-test-bucket} "${args[@]}"

# Run control_client_stall with HNS bucket
go test -v -p 1 -timeout 10m ./tools/integration_tests/emulator_tests/control_client_stall/... --integrationTest --testbucket=test-hns-bucket "${args[@]}"
