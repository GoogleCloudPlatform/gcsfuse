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

function upgrade_gcloud_version() {
  sudo apt-get update
  # Upgrade gcloud version.
  # Kokoro machine's outdated gcloud version prevents the use of the "managed-folders" feature.
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

function create_bucket() {
  bucket_prefix=$1
  # The length of the random string
  length=5
  # Generate the random string
  random_string=$(tr -dc 'a-z0-9' < /dev/urandom | head -c $length)
  bucket_name=$bucket_prefix$random_string
  # We are using gcloud alpha because gcloud storage is giving issues running on Kokoro
  gcloud alpha storage buckets create gs://$bucket_name --project=$project_id --location=$bucket_location --uniform-bucket-level-access
  echo $bucket_name
}

function create_hns_bucket() {
  length=5
  # Generate the random string
  random_string=$(tr -dc 'a-z0-9' < /dev/urandom | head -c $length)
  bucket_name="gcsfuse-e2e-tests-hns-"$random_string
  gcloud alpha storage buckets create gs://$bucket_name --project=$hns_project_id --location=$bucket_location --uniform-bucket-level-access --enable-hierarchical-namespace
  echo "$bucket_name"
}

function run_non_parallel_tests() {
  local exit_code=0
  local -n test_array=$1
  local bucket_name_non_parallel=$2

  for test_dir_np in "${test_array[@]}"
  do
    test_path_non_parallel="./tools/integration_tests/$test_dir_np"
    # Executing integration tests
    GODEBUG=asyncpreemptoff=1 go test $test_path_non_parallel -p 1 --integrationTest -v --testbucket=$bucket_name_non_parallel --testInstalledPackage=$run_e2e_tests_on_package -timeout $integration_test_timeout
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

  for test_dir_p in "${test_array[@]}"
  do
    test_path_parallel="./tools/integration_tests/$test_dir_p"
    # Executing integration tests
    GODEBUG=asyncpreemptoff=1 go test $test_path_parallel -p 1 --integrationTest -v --testbucket=$bucket_name_parallel --testInstalledPackage=$run_e2e_tests_on_package -timeout $integration_test_timeout &
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

function create_flat_buckets() {
  # Test setup
  # Create Bucket for non parallel e2e tests
  # The bucket prefix for the random string
  bucketPrefix="gcsfuse-non-parallel-e2e-tests-group-1-"
  bucket_name_non_parallel_group_1=$(create_bucket $bucketPrefix)
  echo "Bucket name for non parallel tests group - 1: "$bucket_name_non_parallel_group_1

  # Test setup
  # Create Bucket for non parallel e2e tests
  # The bucket prefix for the random string
  bucketPrefix="gcsfuse-non-parallel-e2e-tests-group-2-"
  bucket_name_non_parallel_group_2=$(create_bucket $bucketPrefix)
  echo "Bucket name for non parallel tests group - 2 : "$bucket_name_non_parallel_group_2

  # Create Bucket for parallel e2e tests
  # The bucket prefix for the random string
  bucketPrefix="gcsfuse-parallel-e2e-tests-"
  bucket_name_parallel=$(create_bucket $bucketPrefix)
  echo "Bucket name for parallel tests: "$bucket_name_parallel
}

function run_e2e_tests_for_flat_bucket() {
 # Parallel execution of integration tests located within specified test directories.
 # It aims to improve testing speed by running tests concurrently, while providing basic error reporting.
  echo "Running parallel tests..."
  # Run parallel tests
  run_parallel_tests test_dir_parallel $bucket_name_parallel &
  parallel_tests_pid=$!

 # Non parallel execution of integration tests located within specified test directories.
 echo "Running non parallel tests group-1..."
 run_non_parallel_tests test_dir_non_parallel_group_1 $bucket_name_non_parallel_group_1 &
 non_parallel_tests_pid_group_1=$!
 echo "Running non parallel tests group-2..."
 run_non_parallel_tests test_dir_non_parallel_group_2 $bucket_name_non_parallel_group_2 &
 non_parallel_tests_pid_group_2=$!

 # Wait for all tests to complete.
 wait $parallel_tests_pid
 parallel_tests_exit_code=$?
 wait $non_parallel_tests_pid_group_1
 non_parallel_tests_exit_code_group_1=$?
 wait $non_parallel_tests_pid_group_2
 non_parallel_tests_exit_code_group_2=$?


 if [ $non_parallel_tests_exit_code_group_1 != 0 ] || [ $non_parallel_tests_exit_code_group_2 != 0 ] || [ $parallel_tests_exit_code != 0 ];
 then
   exit 1
 fi
}

function run_e2e_tests_for_hns_bucket(){
   echo "Running tests for HNS bucket"
   run_non_parallel_tests test_dir_hns_group "$hns_bucket_name"
   non_parallel_tests_pid_hns_group=$!

   wait $non_parallel_tests_pid_hns_group
   non_parallel_tests_hns_group_exit_code=$?

   if [ $non_parallel_tests_hns_group_exit_code != 0 ];
   then
     exit 1
   fi
}

#commenting it so cleanup and failure check happens for both
#set -e

function clean_up() {
  # Cleanup
  # Delete bucket after testing.
  local -n buckets=$1
  for bucket in "${buckets[@]}"
    do
      gcloud alpha storage rm --recursive gs://$bucket
    done
}

function clean_up_buckets(){
  flat_buckets=("$bucket_name_parallel" "$bucket_name_non_parallel_group_1" "$bucket_name_non_parallel_group_2")
  hns_buckets=("$hns_bucket_name")

  clean_up flat_buckets
  clean_up hns_buckets
}

function run_tests() {
  run_e2e_tests_for_flat_bucket &
  e2e_tests_flat_bucket_pid=$!

  run_e2e_tests_for_hns_bucket &
  e2e_tests_hns_bucket_pid=$!

  wait $e2e_tests_flat_bucket_pid
  e2e_tests_flat_bucket_status=$?

  wait $e2e_tests_hns_bucket_pid
  e2e_tests_hns_bucket_status=$?

  if [ $e2e_tests_flat_bucket_status != 0 ];
  then
    echo "The e2e tests for flat bucket failed.."
  fi

  if [ $e2e_tests_hns_bucket_status != 0 ];
  then
    echo "The e2e tests for hns bucket failed.."
  fi
}

function main(){
  set -e
  # true or false to run e2e tests on installedPackage
  run_e2e_tests_on_package=$1
  readonly integration_test_timeout=40m
  readonly project_id="gcs-fuse-test-ml"
  readonly hns_project_id="gcs-fuse-test"
  readonly bucket_location="us-west1"

  # Test directory arrays
  test_dir_parallel=(
    "local_file"
    "log_rotation"
    "mounting"
    "read_cache"
    "gzip"
    "write_large_files"
  )

  # These tests never become parallel as it is changing bucket permissions.
  test_dir_non_parallel_group_1=(
    "readonly"
    "managed_folders"
  )

  # These test packages can be configured to run in parallel once they achieve
  # directory independence.
  test_dir_non_parallel_group_2=(
    "explicit_dir"
    "implicit_dir"
    "list_large_dir"
    "operations"
    "read_large_files"
    "rename_dir_limit"
  )

  test_dir_hns_group=(
    "implicit_dir"
    "operations"
  )

  upgrade_gcloud_version

  install_packages

  create_flat_buckets
  hns_bucket_name=$(create_hns_bucket)
  echo "Hns Bucket Created: "$hns_bucket_name

  set +e

  #run integration tests
  run_tests

  # Cleanup
  # Delete bucket after testing.
  clean_up_buckets

  # Removing bin file after testing.
  if [ $run_e2e_tests_on_package != true ];
  then
    sudo rm /usr/local/bin/gcsfuse
  fi
}

#Main method to run script
main "$1"


