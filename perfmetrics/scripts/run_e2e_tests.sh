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
sudo apt-get update

readonly INTEGRATION_TEST_TIMEOUT=40m
readonly PROJECT_ID="gcs-fuse-test-ml"
readonly BUCKET_LOCATION="us-west1"

function upgrade_gcloud_version() {
  gcloud version
  wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
  sudo rm -rf $(which gcloud) && sudo tar xzf gcloud.tar.gz && sudo mv google-cloud-sdk /usr/local
  sudo /usr/local/google-cloud-sdk/install.sh
  export PATH=$PATH:/usr/local/google-cloud-sdk/bin
  echo 'export PATH=$PATH:/usr/local/google-cloud-sdk/bin' >> ~/.bashrc
  gcloud version && rm gcloud.tar.gz && gcloud components update
  sudo /usr/local/google-cloud-sdk/bin/gcloud components install alpha
}

# true or false to run e2e tests on installedPackage
run_e2e_tests_on_package=$1

# Upgrade gcloud version.
# Kokoro machine's outdated gcloud version prevents the use of the "managed-folders" feature.
upgrade_gcloud_version

# e.g. architecture=arm64 or amd64
architecture=$(dpkg --print-architecture)
echo "Installing go-lang 1.21.7..."
wget -O go_tar.tar.gz https://go.dev/dl/go1.21.7.linux-${architecture}.tar.gz -q
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin
# install python3-setuptools tools.
sudo apt-get install -y gcc python3-dev python3-setuptools
# Downloading composite object requires integrity checking with CRC32c in gsutil.
# it requires to install crcmod.
sudo apt install -y python3-crcmod

# Create bucket for integration tests.
function create_bucket() {
  bucketPrefix=$1
  # The length of the random string
  length=5
  # Generate the random string
  random_string=$(tr -dc 'a-z0-9' < /dev/urandom | head -c $length)
  BUCKET_NAME=$bucketPrefix$random_string
  # We are using gcloud alpha because gcloud storage is giving issues running on Kokoro
  gcloud alpha storage buckets create gs://$BUCKET_NAME --project=$PROJECT_ID --location=$BUCKET_LOCATION --uniform-bucket-level-access
  echo $BUCKET_NAME
}

# Non parallel execution of integration tests located within specified test directories.
function run_non_parallel_tests() {
  test_dir_non_parallel=("$@")
  BUCKET_NAME_NON_PARALLEL=$2
  for test_dir_np in "${test_dir_non_parallel[@]}"
  do
    test_path_non_parallel="./tools/integration_tests/$test_dir_np"
    # Executing integration tests
    GODEBUG=asyncpreemptoff=1 go test $test_path_non_parallel -p 1 --integrationTest -v --testbucket=$BUCKET_NAME_NON_PARALLEL --testInstalledPackage=$run_e2e_tests_on_package -timeout $INTEGRATION_TEST_TIMEOUT
    exit_code_non_parallel=$?
    if [ $exit_code_non_parallel != 0 ]; then
      test_fail_np=$exit_code_non_parallel
      echo "test fail in non parallel: " $test_fail_np
    fi
  done
  return $test_fail_np
}

# Parallel execution of integration tests located within specified test directories.
# It aims to improve testing speed by running tests concurrently, while providing basic error reporting.
function run_parallel_tests() {
  test_dir_parallel=("$@")
  BUCKET_NAME_PARALLEL=$2
  for test_dir_p in "${test_dir_parallel[@]}"
  do
    test_path_parallel="./tools/integration_tests/$test_dir_p"
    # Executing integration tests
    GODEBUG=asyncpreemptoff=1 go test $test_path_parallel -p 1 --integrationTest -v --testbucket=$BUCKET_NAME_PARALLEL --testInstalledPackage=$run_e2e_tests_on_package -timeout $INTEGRATION_TEST_TIMEOUT &
    pid=$!  # Store the PID of the background process
    pids+=("$pid")  # Optionally add the PID to an array for later
  done

  # Wait for processes and collect exit codes
  for pid in "${pids[@]}"; do
    wait $pid
    exit_code_parallel=$?
    if [ $exit_code_parallel != 0 ]; then
      test_fail_p=$exit_code_parallel
      echo "test fail in parallel: " $test_fail_p
    fi
  done
  return $test_fail_p
}

# Test setup
# Create Bucket for non parallel e2e tests
# The bucket prefix for the random string
bucketPrefix="gcsfuse-non-parallel-e2e-tests-group-1-"
BUCKET_NAME=$(create_bucket $bucketPrefix)
echo "Bucket name for non parallel tests: "$BUCKET_NAME
BUCKET_NAME_NON_PARALLEL_GROUP_1=$BUCKET_NAME
# Test directory array
# These tests never become parallel as it is changing bucket permissions.
test_dir_non_parallel_group_1=(
  "readonly"
  "managed_folders"
)

# Test setup
# Create Bucket for non parallel e2e tests
# The bucket prefix for the random string
bucketPrefix="gcsfuse-non-parallel-e2e-tests-group-2-"
echo "Bucket name for non parallel tests - 2 : "$BUCKET_NAME
BUCKET_NAME=$(create_bucket $bucketPrefix)
BUCKET_NAME_NON_PARALLEL_GROUP_2=$BUCKET_NAME
# These test packages can be configured to run in parallel once they achieve
# directory independence.
# Test directory array
test_dir_non_parallel_group_2=(
  "explicit_dir"
  "implicit_dir"
  "list_large_dir"
  "operations"
  "read_large_files"
  "rename_dir_limit"
)

# Create Bucket for parallel e2e tests
# The bucket prefix for the random string
bucketPrefix="gcsfuse-parallel-e2e-tests-"
BUCKET_NAME=$(create_bucket $bucketPrefix)
echo "Bucket name for parallel tests: "$BUCKET_NAME
BUCKET_NAME_PARALLEL=$BUCKET_NAME
# Test directory array
test_dir_parallel=(
  "local_file"
  "log_rotation"
  "mounting"
  "read_cache"
  "gzip"
  "write_large_files"
)

# Run tests
test_fail_p=0
test_fail_np=0
test_fail_np_group_1=0
test_fail_np_group_2=0
set +e

echo "Running parallel tests..."
# Run parallel tests
run_parallel_tests ${test_dir_parallel[@]} $$BUCKET_NAME_PARALLEL &
my_process_p=$!
echo "Running non parallel tests..."
# Run non parallel tests
run_non_parallel_tests ${test_dir_non_parallel_group_1[@]} $BUCKET_NAME_NON_PARALLEL_GROUP_1 &
my_process_np_group_1=$!
# Run non parallel tests
run_non_parallel_tests ${test_dir_non_parallel_group_2[@]} $BUCKET_NAME_NON_PARALLEL_GROUP_2 &
my_process_np_group_2=$!
wait $my_process_p
test_fail_p=$?
wait $my_process_np_group_1
test_fail_np_group_1=$?
wait $my_process_np_group_2
test_fail_np_group_2=$?
set -e

# Cleanup
# Delete bucket after testing.
gcloud alpha storage rm --recursive gs://$BUCKET_NAME_PARALLEL/
gcloud alpha storage rm --recursive gs://$BUCKET_NAME_NON_PARALLEL_GROUP_1/
gcloud alpha storage rm --recursive gs://$BUCKET_NAME_NON_PARALLEL_GROUP_2/

# Removing bin file after testing.
if [ $run_e2e_tests_on_package != true ];
then
  sudo rm /usr/local/bin/gcsfuse
fi
if [ $test_fail_np_group_1 != 0 ] || [ $test_fail_np_group_2 != 0 ] || [ $test_fail_p != 0 ];
then
  echo "The tests failed."
  exit 1
fi
