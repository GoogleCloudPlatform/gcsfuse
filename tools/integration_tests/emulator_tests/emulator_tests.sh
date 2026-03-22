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

# Script Usage Documentation
usage() {
  echo "Usage: $0 [options]"
  echo "Options:"
  echo "    --test-installed-package                     Test installed gcsfuse package. (Default: false)"
  echo "    --gcsfuse_prebuilt_dir   <path>              Path to pre-built gcsfuse binary for testing (e.g. /path/to/gcsfuse/binary)"
  echo "                                                 This option is mutually exclusive with --test-installed-package. (Default: "")"
  echo "    --help                                       Display this help and exit."
  exit "$1"
}

# Logging Helpers
log_info() {
  echo "[INFO] $(date +"%Y-%m-%d %H:%M:%S"): $1"
}

log_error() {
  echo "[ERROR] $(date +"%Y-%m-%d %H:%M:%S"): $1"
}

TEST_INSTALLED_PACKAGE=false
GCSFUSE_PREBUILT_DIR=""
# Define options for getopt
# A long option name followed by a colon indicates it requires an argument.
LONG=test-installed-package,gcsfuse_prebuilt_dir:,help

# Parse the options using getopt
# --options "" specifies that there are no short options.
if ! PARSED=$(getopt --options "" --longoptions "$LONG" --name "$0" -- "$@"); then
    usage 1
fi

# Read the parsed options back into the positional parameters.
eval set -- "$PARSED"

# Loop through the options and assign values to our variables
while (( $# >= 1 )); do
    case "$1" in
        --test-installed-package)
            TEST_INSTALLED_PACKAGE=true
            shift 
            ;;
        --gcsfuse_prebuilt_dir)
            GCSFUSE_PREBUILT_DIR="$2"
            shift 2
            ;;
        --help)
            usage 0
            ;;
        --)
            shift
            break
            ;;
        *)
            log_error "Unrecognized arguments [$*]."
            usage 1
            ;;
    esac
done

uname=$(uname -m)
if [ $uname == "aarch64" ];then
  # TODO: Remove this when we have an ARM64 image for the storage test bench.(b/384388821)
  log_info "These tests will not run for arm64 machine..."
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
if sudo docker ps > /dev/null 2>&1; then
  echo "Docker is already installed and usable. Skipping installation steps."
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

wait_for_emulator() {
  local timeout=30
  local count=0
  echo "Waiting for emulator to be ready..."
  while [ $count -lt $timeout ]; do
    if curl -s "$STORAGE_EMULATOR_HOST/storage/v1/b?project=test-project" > /dev/null; then
      echo "Emulator is ready!"
      return 0
    fi
    sleep 1
    count=$((count+1))
  done
  echo "Emulator failed to become ready after $timeout seconds."
  return 1
}

# Start the testbench
sudo docker run --name $CONTAINER_NAME -e GUNICORN_CMD_ARGS="--timeout 600" --rm -d $DOCKER_NETWORK $DOCKER_IMAGE
echo "Docker logs are saved at: $(pwd)/emulator_container.log"
sudo docker logs -f $CONTAINER_NAME > emulator_container.log 2>&1 &

wait_for_emulator

# Stop the testbench & cleanup environment variables
function cleanup() {
    echo "Cleanup testbench"
    sudo docker stop $CONTAINER_NAME
    unset STORAGE_EMULATOR_HOST;
    echo "Printing Emulator Container Logs..."
    cat emulator_container.log
}
trap cleanup EXIT

# Create the JSON file to create bucket
cat << EOF > test.json
{"name":"test-bucket"}
EOF

# Execute the curl command to create bucket on storagetestbench server.
if ! curl -X POST --data-binary @test.json \
    -H "Content-Type: application/json" \
    "$STORAGE_EMULATOR_HOST/storage/v1/b?project=test-project"; then
  echo "Failed to create bucket test-bucket"
  exit 1
fi
rm test.json

# Run all emulator test packages sequentially.
go test -p 1 ./tools/integration_tests/emulator_tests/... --integrationTest -v --testbucket=test-bucket -timeout 20m --testInstalledPackage=$TEST_INSTALLED_PACKAGE --gcsfuse_prebuilt_dir=$GCSFUSE_PREBUILT_DIR
