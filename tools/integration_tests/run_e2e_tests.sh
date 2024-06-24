#!/bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
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

# This will stop execution when any command will have non-zero status.

# true or false to run e2e tests on installedPackage
RUN_E2E_TESTS_ON_PACKAGE=$1

# Pass "true" to skip few non-essential tests.
# By default, this script runs all the integration tests.
SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE=$2
BUCKET_LOCATION=$3
# Pass "true" to run e2e tests on TPC endpoint.
# The default value will be false.
RUN_TEST_ON_TPC_ENDPOINT=false
if [ $4 != "" ]; then
  RUN_TEST_ON_TPC_ENDPOINT=$4
fi
INTEGRATION_TEST_TIMEOUT=60m

if [ "$#" -lt 3 ]
then
  echo "Incorrect number of arguments passed, please refer to the script and pass the three arguments required..."
  exit 1
fi

if [ "$SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE" == true ]; then
  GO_TEST_SHORT_FLAG="-short"
  echo "Setting the flag to skip few un-important integration tests."
  INTEGRATION_TEST_TIMEOUT=40m
  echo "Changing the integration test timeout to: $INTEGRATION_TEST_TIMEOUT"
fi

readonly RANDOM_STRING_LENGTH=5
# Test directory arrays
TEST_DIR_PARALLEL=(
  "local_file"
  "log_rotation"
  "mounting"
  "read_cache"
  "gzip"
  "write_large_files"
  "list_large_dir"
  "rename_dir_limit"
  "read_large_files"
  "explicit_dir"
  "implicit_dir"
  "interrupt"
  "operations"
  "log_content"
)
# These tests never become parallel as it is changing bucket permissions.
TEST_DIR_NON_PARALLEL=(
  "readonly"
  "managed_folders"
  "readonly_creds"
)


TEST_DIR_HNS_GROUP=(
  "implicit_dir"
  "operations"
)

# Create a temporary file to store the log file name.
TEST_LOGS_FILE=$(mktemp)

function upgrade_gcloud_version() {
  sudo apt-get update
  # Upgrade gcloud version.
  # Kokoro machine's outdated gcloud version prevents the use of the "managed-folders" feature.
  gcloud version
  wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
  sudo tar xzf gcloud.tar.gz && sudo cp -r google-cloud-sdk /usr/local && sudo rm -r google-cloud-sdk
  sudo /usr/local/google-cloud-sdk/install.sh
  export PATH=/usr/local/google-cloud-sdk/bin:$PATH
  echo 'export PATH=/usr/local/google-cloud-sdk/bin:$PATH' >> ~/.bashrc
  gcloud version && rm gcloud.tar.gz
  sudo /usr/local/google-cloud-sdk/bin/gcloud components update
  sudo /usr/local/google-cloud-sdk/bin/gcloud components install alpha
}

function install_packages() {
  # e.g. architecture=arm64 or amd64
  architecture=$(dpkg --print-architecture)
  echo "Installing go-lang 1.22.4..."
  wget -O go_tar.tar.gz https://go.dev/dl/go1.22.4.linux-${architecture}.tar.gz -q
  sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
  export PATH=$PATH:/usr/local/go/bin
  # install python3-setuptools tools.
  sudo apt-get install -y gcc python3-dev python3-setuptools
  # Downloading composite object requires integrity checking with CRC32c in gsutil.
  # it requires to install crcmod.
  sudo apt install -y python3-crcmod
}

function create_bucket() {
  bucket_prefix=$1
  local -r project_id="gcs-fuse-test-ml"
  # Generate bucket name with random string
  bucket_name=$bucket_prefix$(tr -dc 'a-z0-9' < /dev/urandom | head -c $RANDOM_STRING_LENGTH)
  # We are using gcloud alpha because gcloud storage is giving issues running on Kokoro
  gcloud alpha storage buckets create gs://$bucket_name --project=$project_id --location=$BUCKET_LOCATION --uniform-bucket-level-access
  echo $bucket_name
}

function create_hns_bucket() {
  local -r hns_project_id="gcs-fuse-test"
  # Generate bucket name with random string.
  # Adding prefix `golang-grpc-test` to white list the bucket for grpc
  # so that we can run grpc related e2e tests.
  bucket_name="golang-grpc-test-gcsfuse-e2e-tests-hns-"$(tr -dc 'a-z0-9' < /dev/urandom | head -c $RANDOM_STRING_LENGTH)
  gcloud alpha storage buckets create gs://$bucket_name --project=$hns_project_id --location=$BUCKET_LOCATION --uniform-bucket-level-access --enable-hierarchical-namespace
  echo "$bucket_name"
}

function run_non_parallel_tests() {
  local exit_code=0
  local -n test_array=$1
  local bucket_name_non_parallel=$2

  for test_dir_np in "${test_array[@]}"
  do
    test_path_non_parallel="./tools/integration_tests/$test_dir_np"
    # To make it clear whether tests are running on a flat or HNS bucket, We kept the log file naming
    # convention to include the bucket name as a suffix (e.g., package_name_bucket_name).
    local log_file="/tmp/${test_dir_np}_${bucket_name_non_parallel}.log"
    echo $log_file >> $TEST_LOGS_FILE

    # Executing integration tests
    GODEBUG=asyncpreemptoff=1 go test $test_path_non_parallel -p 1 $GO_TEST_SHORT_FLAG --integrationTest -v --testbucket=$bucket_name_non_parallel --testInstalledPackage=$RUN_E2E_TESTS_ON_PACKAGE -timeout $INTEGRATION_TEST_TIMEOUT > "$log_file" 2>&1
    exit_code_non_parallel=$?
    if [ $exit_code_non_parallel != 0 ]; then
      exit_code=$exit_code_non_parallel
      echo "test fail in non parallel on package: " $test_dir_np
    fi
  done
  return $exit_code
}

function run_parallel_tests() {
  local exit_code=0
  local -n test_array=$1
  local bucket_name_parallel=$2
  local pids=()

  for test_dir_p in "${test_array[@]}"
  do
    test_path_parallel="./tools/integration_tests/$test_dir_p"
    # To make it clear whether tests are running on a flat or HNS bucket, We kept the log file naming
    # convention to include the bucket name as a suffix (e.g., package_name_bucket_name).
    local log_file="/tmp/${test_dir_p}_${bucket_name_parallel}.log"
    echo $log_file >> $TEST_LOGS_FILE
    # Executing integration tests
    GODEBUG=asyncpreemptoff=1 go test $test_path_parallel $GO_TEST_SHORT_FLAG -p 1 --integrationTest -v --testbucket=$bucket_name_parallel --testInstalledPackage=$RUN_E2E_TESTS_ON_PACKAGE -timeout $INTEGRATION_TEST_TIMEOUT > "$log_file" 2>&1 &
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

function print_test_logs() {
  readarray -t test_logs_array < "$TEST_LOGS_FILE"
  rm "$TEST_LOGS_FILE"
  for test_log_file in "${test_logs_array[@]}"
  do
    log_file=${test_log_file}
    if [ -f "$log_file" ]; then
      echo "=== Log for ${test_log_file} ==="
      cat "$log_file"
      echo "========================================="
    fi
  done
}

function run_e2e_tests_for_flat_bucket() {
  # Adding prefix `golang-grpc-test` to white list the bucket for grpc so that
  # we can run grpc related e2e tests.
  bucketPrefix="golang-grpc-test-gcsfuse-non-parallel-e2e-tests-"
  bucket_name_non_parallel=$(create_bucket $bucketPrefix)
  echo "Bucket name for non parallel tests: "$bucket_name_non_parallel

  bucketPrefix="golang-grpc-test-gcsfuse-parallel-e2e-tests-"
  bucket_name_parallel=$(create_bucket $bucketPrefix)
  echo "Bucket name for parallel tests: "$bucket_name_parallel

  echo "Running parallel tests..."
  run_parallel_tests TEST_DIR_PARALLEL $bucket_name_parallel &
  parallel_tests_pid=$!

 echo "Running non parallel tests ..."
 run_non_parallel_tests TEST_DIR_NON_PARALLEL $bucket_name_non_parallel &
 non_parallel_tests_pid=$!

 # Wait for all tests to complete.
 wait $parallel_tests_pid
 parallel_tests_exit_code=$?
 wait $non_parallel_tests_pid
 non_parallel_tests_exit_code=$?

 flat_buckets=("$bucket_name_parallel" "$bucket_name_non_parallel")
 clean_up flat_buckets

 if [ $non_parallel_tests_exit_code != 0 ] || [ $parallel_tests_exit_code != 0 ];
 then
   return 1
 fi
 return 0
}

function run_e2e_tests_for_hns_bucket(){
   hns_bucket_name=$(create_hns_bucket)
   echo "Hns Bucket Created: "$hns_bucket_name

   echo "Running tests for HNS bucket"
   run_non_parallel_tests TEST_DIR_HNS_GROUP "$hns_bucket_name"
   non_parallel_tests_hns_group_exit_code=$?

   hns_buckets=("$hns_bucket_name")
   clean_up hns_buckets

   if [ $non_parallel_tests_hns_group_exit_code != 0 ];
   then
     return 1
   fi
   return 0
}

function run_e2e_tests_for_tpc() {
  # Clean bucket before testing.
  gcloud storage rm -r gs://gcsfuse-e2e-tests-tpc/**

  # Run Operations e2e tests in TPC to validate all the functionality.
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/... --testOnTPCEndPoint=$RUN_TEST_ON_TPC_ENDPOINT $GO_TEST_SHORT_FLAG -p 1 --integrationTest -v --testbucket=gcsfuse-e2e-tests-tpc --testInstalledPackage=$RUN_E2E_TESTS_ON_PACKAGE -timeout $INTEGRATION_TEST_TIMEOUT
  exit_code=$?

  set -e

  # Delete data after testing.
  gcloud storage rm -r gs://gcsfuse-e2e-tests-tpc/**

  if [ $exit_code != 0 ];
   then
     echo "The tests failed."
  fi
  exit $exit_code
}

#commenting it so cleanup and failure check happens for both
#set -e

function clean_up() {
  # Cleanup
  # Delete bucket after testing.
  local -n buckets=$1
  for bucket in "${buckets[@]}"
    do
      # Empty bucket name may cause deletions of all the buckets.
      if [ "$bucket" != "" ];
      then
        gcloud alpha storage rm --recursive gs://$bucket
      fi
    done
}

function main(){
  set -e

  upgrade_gcloud_version

  install_packages

  set +e

  # Run tpc test and exit in case RUN_TEST_ON_TPC_ENDPOINT is true.
  if [ $RUN_TEST_ON_TPC_ENDPOINT == true ]; then
       run_e2e_tests_for_tpc
  fi

  #run integration tests
  run_e2e_tests_for_hns_bucket &
  e2e_tests_hns_bucket_pid=$!

  run_e2e_tests_for_flat_bucket &
  e2e_tests_flat_bucket_pid=$!

  wait $e2e_tests_flat_bucket_pid
  e2e_tests_flat_bucket_status=$?

  wait $e2e_tests_hns_bucket_pid
  e2e_tests_hns_bucket_status=$?

  set -e

  print_test_logs

  if [ $e2e_tests_flat_bucket_status != 0 ] && [ $e2e_tests_hns_bucket_status != 0 ];
  then
    echo "The e2e tests for both flat and hns bucket failed.."
    exit 1
  fi

  if [ $e2e_tests_flat_bucket_status != 0 ];
  then
    echo "The e2e tests for flat bucket failed.."
    exit 1
  fi

  if [ $e2e_tests_hns_bucket_status != 0 ];
  then
    echo "The e2e tests for hns bucket failed.."
    exit 1
  fi
}

#Main method to run script
main
