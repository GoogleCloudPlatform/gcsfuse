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

# Only run on Go 1.17+
min_minor_ver=17

v=`go version | { read _ _ v _; echo ${v#go}; }`
comps=(${v//./ })
minor_ver=${comps[1]}

if [ "$minor_ver" -lt "$min_minor_ver" ]; then
    echo minor version $minor_ver, skipping
    exit 0
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
docker pull $DOCKER_IMAGE

# Start the testbench
docker run --name $CONTAINER_NAME --rm -d $DOCKER_NETWORK $DOCKER_IMAGE
echo "Running the Cloud Storage testbench: $STORAGE_EMULATOR_HOST"
sleep 5

# Stop the testbench & cleanup environment variables
function cleanup() {
    echo "Cleanup testbench"
    docker stop $CONTAINER_NAME
    unset STORAGE_EMULATOR_HOST;
}
trap cleanup EXIT

# Create the JSON payload file
cat << EOF > test.json
{"name":"test-bucket"}
EOF

# Execute the curl command to create bucket on storagetestbench server.
curl -X POST --data-binary @test.json \
    -H "Content-Type: application/json" \
    "$STORAGE_EMULATOR_HOST/storage/v1/b?project=test-project"
rm test.json
# Define an associative array to store config file and corresponding test name pairs
# e.g. [write_stall.yaml, TestWriteStall]
declare -A config_test_pairs=(

)

# Loop through the array of config file and test name pairs
for config_file in "${!config_test_pairs[@]}"; do
  cd proxy_server

  test_name="${config_test_pairs[$config_file]}"
  echo "Running proxy with config file: $config_file and test suite: $test_name"

  # Run the proxy server in the background with the current config file
  nohup go run . --debug --config-path ./configs/$config_file &
  proxy_pid=$!
  echo "Proxy process: $proxy_pid"

  # Move to the parent directory (assuming your tests are in the parent directory)
  cd ..

  # Run tests with the STORAGE_EMULATOR_HOST environment variable set and specific test suite
  STORAGE_EMULATOR_HOST="http://localhost:8020" go test --integrationTest -v --testbucket=test-bucket -timeout 10m -run $test_name
  # Kill the proxy process
  kill -2 $proxy_pid

  # Find and kill any processes still listening on port 8020
  p_listing_8020=$(lsof -i :8020 | awk '{print $2}' | tail -n +2)
  echo "Process listning on 8020: $p_listing_8020"
  kill -2 $p_listing_8020
done