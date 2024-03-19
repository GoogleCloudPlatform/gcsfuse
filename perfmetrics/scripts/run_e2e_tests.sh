#!/bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This will stop execution when any command will have non-zero status.

set -e

# true or false to run e2e tests on installedPackage
RUN_E2E_TESTS_ON_PACKAGE=$1
readonly INTEGRATION_TEST_TIMEOUT=40m
readonly PROJECT_ID="gcs-fuse-test-ml"
readonly BUCKET_LOCATION="us-west1"

# Test directory arrays
TEST_DIR_PARALLEL=(
  "local_file"
  "log_rotation"
  "mounting"
  "read_cache"
  "gzip"
  "write_large_files"
)

# These tests never become parallel as it is changing bucket permissions.
TEST_DIR_NON_PARALLEL_GROUP_1=(
  "readonly"
  "managed_folders"
)

# These test packages can be configured to run in parallel once they achieve
# directory independence.
TEST_DIR_NON_PARALLEL_GROUP_2=(
  "explicit_dir"
  "implicit_dir"
  "list_large_dir"
  "operations"
  "read_large_files"
  "rename_dir_limit"
)

function upgrade_gcloud_version() {
  gcloud version
  wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
  sudo tar xzf gcloud.tar.gz && sudo mv google-cloud-sdk /usr/local
  sudo /usr/local/google-cloud-sdk/install.sh
  export PATH=/usr/local/google-cloud-sdk/bin:$PATH
  echo 'export PATH=/usr/local/google-cloud-sdk/bin:$PATH' >> ~/.bashrc
  gcloud version && rm gcloud.tar.gz
  sudo /usr/local/google-cloud-sdk/bin/gcloud components update
  sudo /usr/local/google-cloud-sdk/bin/gcloud components install alpha
}

# Install packages
function install_packages() {
  # e.g. architecture=arm64 or amd64
  architecture=$(dpkg --print-architecture)
  echo "Installing go-lang 1.22.1..."
  wget -O go_tar.tar.gz https://go.dev/dl/go1.22.1.linux-${architecture}.tar.gz -q
  sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
  export PATH=$PATH:/usr/local/go/bin
  # install python3-setuptools tools.
  sudo apt-get install -y gcc python3-dev python3-setuptools
  # Downloading composite object requires integrity checking with CRC32c in gsutil.
  # it requires to install crcmod.
  sudo apt install -y python3-crcmod
}

# Create bucket for integration tests.
function create_bucket() {
  bucket_prefix=$1
  # The length of the random string
  length=5
  # Generate the random string
  random_string=$(tr -dc 'a-z0-9' < /dev/urandom | head -c $length)
  bucket_name=$bucket_prefix$random_string
  # We are using gcloud alpha because gcloud storage is giving issues running on Kokoro
  gcloud alpha storage buckets create gs://$bucket_name --project=$PROJECT_ID --location=$BUCKET_LOCATION --uniform-bucket-level-access
  echo $bucket_name
}

# Non parallel execution of integration tests located within specified test directories.
function run_non_parallel_tests() {
  local exit_code=0
  local -n test_array=$1
  local bucket_name_non_parallel=$2

  for test_dir_np in "${test_array[@]}"
  do
    test_path_non_parallel="./tools/integration_tests/$test_dir_np"
    # Executing integration tests
    GODEBUG=asyncpreemptoff=1 go test $test_path_non_parallel -p 1 --integrationTest -v --testbucket=$bucket_name_non_parallel --testInstalledPackage=$RUN_E2E_TESTS_ON_PACKAGE -timeout $INTEGRATION_TEST_TIMEOUT
    exit_code_non_parallel=$?
    if [ $exit_code_non_parallel != 0 ]; then
      exit_code=$exit_code_non_parallel
      echo "test fail in non parallel on package: " $test_dir_np
    fi
  done
  return $exit_code
}

# Parallel execution of integration tests located within specified test directories.
# It aims to improve testing speed by running tests concurrently, while providing basic error reporting.
function run_parallel_tests() {
  local exit_code=0
  local -n test_array=$1
  local bucket_name_parallel=$2

  for test_dir_p in "${test_array[@]}"
  do
    test_path_parallel="./tools/integration_tests/$test_dir_p"
    # Executing integration tests
    GODEBUG=asyncpreemptoff=1 go test $test_path_parallel -p 1 --integrationTest -v --testbucket=$bucket_name_parallel --testInstalledPackage=$RUN_E2E_TESTS_ON_PACKAGE -timeout $INTEGRATION_TEST_TIMEOUT &
    pid=$!  # Store the PID of the background process
    pids+=("$pid")  # Optionally add the PID to an array for later
  done

  # Wait for processes and collect exit codes
  for pid in "${pids[@]}"; do
    wait $pid
    exit_code_parallel=$?
    if [ $exit_code_parallel != 0 ]; then
      exit_code=$exit_code_parallel
      echo "test fail in parallel on package: " $test_dir_p
    fi
  done
  return $exit_code
}

sudo apt-get update
# Upgrade gcloud version.
# Kokoro machine's outdated gcloud version prevents the use of the "managed-folders" feature.
upgrade_gcloud_version
install_packages

# Test setup
# Create Bucket for non parallel e2e tests
# The bucket prefix for the random string
bucket_prefix="gcsfuse-non-parallel-e2e-tests-group-1-"
bucket_name_non_parallel_group_1=$(create_bucket $bucket_prefix)
echo "Bucket name for non parallel tests group - 1: "$bucket_name_non_parallel_group_1

# Test setup
# Create Bucket for non parallel e2e tests
# The bucket prefix for the random string
bucket_prefix="gcsfuse-non-parallel-e2e-tests-group-2-"
bucket_name_non_parallel_group_2=$(create_bucket $bucket_prefix)
echo "Bucket name for non parallel tests group - 2 : "$bucket_name_non_parallel_group_2

# Create Bucket for parallel e2e tests
# The bucket prefix for the random string
bucket_prefix="gcsfuse-parallel-e2e-tests-"
bucket_name_parallel=$(create_bucket $bucket_prefix)
echo "Bucket name for parallel tests: "$bucket_name_parallel

# Run tests
set +e
echo "Running parallel tests..."
# Run parallel tests
run_parallel_tests TEST_DIR_PARALLEL $bucket_name_parallel &
parallel_tests_pid=$!

# Run non parallel tests
echo "Running non parallel tests group-1..."
run_non_parallel_tests TEST_DIR_NON_PARALLEL_GROUP_1 $bucket_name_non_parallel_group_1 &
non_parallel_tests_pid_group_1=$!
echo "Running non parallel tests group-2..."
run_non_parallel_tests TEST_DIR_NON_PARALLEL_GROUP_2 $bucket_name_non_parallel_group_2 &
non_parallel_tests_pid_group_2=$!

# Wait for all tests to complete.
wait $parallel_tests_pid
parallel_tests_exit_code=$?
wait $non_parallel_tests_pid_group_1
non_parallel_tests_exit_code_group_1=$?
wait $non_parallel_tests_pid_group_2
non_parallel_tests_exit_code_group_2=$?
set -e

# Cleanup
# Delete bucket after testing.
gcloud alpha storage rm --recursive gs://$bucket_name_parallel/
gcloud alpha storage rm --recursive gs://$bucket_name_non_parallel_group_1/
gcloud alpha storage rm --recursive gs://$bucket_name_non_parallel_group_2/

# Removing bin file after testing.
if [ $RUN_E2E_TESTS_ON_PACKAGE != true ];
then
  sudo rm /usr/local/bin/gcsfuse
fi
if [ $non_parallel_tests_exit_code_group_1 != 0 ] || [ $non_parallel_tests_exit_code_group_2 != 0 ] || [ $parallel_tests_exit_code != 0 ];
then
  echo "The tests failed."
  exit 1
fi
