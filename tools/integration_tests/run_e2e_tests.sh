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

# Required GO version for this script.
GO_VERSION="go1.24.0"
PROJECT_ID="gcs-fuse-test-ml"
HNS_PROJECT_ID="gcs-fuse-test"
BUCKET_PREFIX="gcs-fuse-e2e-tests"

# --- Default values for optional arguments ---
RUN_TEST_ON_TPC_ENDPOINT=false
RUN_TESTS_WITH_PRESUBMIT_FLAG=false
RUN_TESTS_WITH_ZONAL_BUCKET=false

# --------- Constants ---------
readonly LOG_LOCK_FILE=$(mktemp /tmp/logging_lock.XXXXXX)
readonly BUCKET_CREATION_LOCK_FILE=$(mktemp /tmp/bucket_creation_lock.XXXXXX)
readonly INTEGRATION_TEST_PACKAGE_DIR="./tools/integration_tests"
readonly INTEGRATION_TEST_TIMEOUT_IN_MINS=90


# --- Usage function ---
usage() {
  echo "Usage: $0 <test_installed_package> <skip_non_essential> <bucket_location> [is_presubmit_run] [run_on_tpc_endpoint] [test_with_zonal_bucket]"
  echo ""
  echo "Required Arguments (Positional):"
  echo "  <test_installed_package>           (Argument 1) Set to 'true' to run e2e tests on the installed GCSFuse package, 'false' otherwise."
  echo "  <skip_non_essential>               (Argument 2) Set to 'true' to skip non-essential integration tests, 'false' to run all."
  echo "  <bucket_location>                  (Argument 3) The bucket location for the tests (e.g., 'us-west1')."
  echo ""                                   
  echo "Optional Arguments (Positional):"
  echo "  [is_presubmit_run]                 (Argument 4) Set to 'true' if this is a presubmit run (skips some tests and lowers timeout)."
  echo "                                     (Default: ${RUN_TESTS_WITH_PRESUBMIT_FLAG})"
  echo "  [run_on_tpc_endpoint]              (Argument 5) Set to 'true' to run operations e2e tests on TPC endpoint."
  echo "                                     (Default: ${RUN_TEST_ON_TPC_ENDPOINT})"
  echo "  [use_zonal_bucket]                 (Argument 6) Set to 'true' to run e2e tests on zonal bucket."
  echo ""
  echo "Examples:"
  echo "  # Run all e2e and integration tests in us-east1 on installed GCSFuse package"
  echo "  $0 true false us-east1"
  echo ""
  echo "  # Run e2e tests, skip non-essential, in us-west1 on installed GCSFuse package"
  echo "  $0 true true us-west1"
  echo ""
  echo "  # Run e2e tests, skip non-essential, in us-central1 for presubmit"
  echo "  $0 false false us-central1 true"
  echo ""
  exit 1
}

# --- Argument Parsing and Assignment using shift ---

# Check for minimum required arguments
if [ "$#" -lt 3 ]; then
  echo "Error: Missing required arguments."
  usage
fi

# Check for total number of arguments
if [ "$#" -gt 6 ]; then
  echo "Error: Too many arguments."
  usage
fi

TEST_INSTALLED_PACKAGE="$1"
shift
SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE="$1"
shift
BUCKET_LOCATION="$1"
shift
if [ -n "$1" ]; then
  RUN_TESTS_WITH_PRESUBMIT_FLAG="$1"
  shift
fi
if [ -n "$1" ]; then
  RUN_TEST_ON_TPC_ENDPOINT="$1"
  shift
fi
if [ -n "$1" ]; then
  RUN_TESTS_WITH_ZONAL_BUCKET="$1"
  shift
fi

# Tests Packages which can be run in parallel.
PARALLEL_TEST_PACKAGES=(
  "monitoring"
  "local_file"
  "log_rotation"
  "mounting"
  "read_cache"
  # "grpc_validation"
  "gzip"
  "write_large_files"
  "list_large_dir"
  "rename_dir_limit"
  "read_large_files"
  "explicit_dir"
  "implicit_dir"
  "interrupt"
  "operations"
  "kernel_list_cache"
  "concurrent_operations"
  "benchmarking"
  "mount_timeout"
  "stale_handle"
  "negative_stat_cache"
  "streaming_writes"
)

# These packages which can only be run in sequential.
SEQUENTIAL_TEST_PACKAGES=(
  "readonly"
  "managed_folders"
  "readonly_creds"
)

# Tests Packages which can be run in parallel on zonal buckets.
PARALLEL_TEST_PACKAGES_FOR_ZB=(
  "benchmarking"
  "explicit_dir"
  "gzip"
  "implicit_dir"
  "interrupt"
  "kernel_list_cache"
  "local_file"
  "log_rotation"
  "monitoring"
  "mount_timeout"
  "mounting"
  "negative_stat_cache"
  "operations"
  "read_cache"
  "read_large_files"
  "rename_dir_limit"
  "stale_handle"
  "streaming_writes"
  "write_large_files"
  "unfinalized_object"
)

# These packages which can only be run in sequential on zonal buckets.
SEQUENTIAL_TEST_PACKAGES_FOR_ZB=(
  "concurrent_operations"
  "list_large_dir"
  "managed_folders"
  "readonly"
  "readonly_creds"
)

# --- Locking Functions ---

# acquire_lock: Acquires exclusive lock or exits script on failure.
# Args: $1 = path to lock file.
function acquire_lock() {
  local lock_file="$1"
  [[ -z "$lock_file" ]] && { echo "acquire_lock: Lock file path is required."; exit 1; }
  exec 200>"$lock_file" || { echo "Could not open lock file $lock_file."; exit 1; }
  flock -x 200 || { log_error "Failed to acquire lock on $lock_file."; exit 1; }
  return 0
}

# release_lock: Releases lock or exits script on failure.
# Args: $1 = path to lock file
function release_lock() {
  local lock_file="$1"
  [[ -z "$lock_file" ]] && { echo "release_lock: Lock file path is required."; exit 1; }
  [[ -e "/proc/self/fd/200" || -L "/proc/self/fd/200" ]] && exec 200>&- || { echo "Lock file descriptor (FD 200) not open for $lock_file. Possible previous error or double release."; exit 1; } # FD not open or close failed
  return 0
}

# Function to get current timestamp in a consistent format
_get_timestamp() {
  date +"%Y-%m-%d %H:%M:%S"
}

function log_info() {
  acquire_lock "$LOG_LOCK_FILE"
  echo "[INFO] $(_get_timestamp) - $$1"
  release_lock "$LOG_LOCK_FILE"
}

function log_error() {
  acquire_lock "$LOG_LOCK_FILE"
  echo "[ERROR] $(_get_timestamp) - $1"
  release_lock "$LOG_LOCK_FILE"
}

# Stores the names of buckets created using create_bucket function.
BUCKET_NAMES=()

# shellcheck disable=SC2317
function create_bucket() {
  local bucket_type="$1"
  local uuid
  uuid=$(uuidgen)
  if [[ -z "$uuid" ]]; then 
    log_error "Unable to generate random UUID for bucket name"
    return 1
  fi
  local bucket_name="${BUCKET_PREFIX}-${bucket_type}-${uuid}"
  local cmd
  if [[ "$bucket_type" == "flat" ]]; then
    cmd="gcloud alpha storage buckets create gs://${bucket_name} --project=${PROJECT_ID} --location=${BUCKET_LOCATION} --uniform-bucket-level-access > /dev/null 2>&1"
  elif [[ "$bucket_type" == "hns" ]]; then
    cmd="gcloud alpha storage buckets create gs://${bucket_name} --project=${HNS_PROJECT_ID} --location=${BUCKET_LOCATION} --uniform-bucket-level-access --enable-hierarchical-namespace > /dev/null 2>&1"
  elif [[ "$bucket_type" == "zonal" ]]; then 
    cmd="gcloud alpha storage buckets create gs://${bucket_name} --project=${PROJECT_ID} --location=${BUCKET_LOCATION} --placement=${BUCKET_LOCATION}-a --default-storage-class=RAPID --uniform-bucket-level-access --enable-hierarchical-namespace > /dev/null 2>&1"
  else
    log_error "Invalid Bucket Type [${bucket_type}]. Supported Types [flat, hns, zonal]"
    return 1
  fi
  acquire_lock "$BUCKET_CREATION_LOCK_FILE"
  if ! eval "$cmd"; then
    log_error "Unable to create bucket [${bucket_name}]"
    sleep 3
    release_lock "$BUCKET_CREATION_LOCK_FILE"
    return 1
  fi
  sleep 3
  release_lock "$BUCKET_CREATION_LOCK_FILE"
  BUCKET_NAMES+=("$bucket_name")
  echo "$bucket_name"
  return 0
}

# shellcheck disable=SC2317
function delete_bucket() {
  local bucket="$1"
  if ! gcloud -q storage rm -r "gs://${bucket}"; then
    log_error "Unable to delete bucket [${bucket}]"
    return 1
  fi
  return 0
}

# shellcheck disable=SC2317
function delete_buckets() {
    local buckets=("$@")
    if ! run_parallel "delete_bucket @" "${buckets[@]}"; then
      log_error "Failed to delete all buckets"
    else
      log_info "Successfully deleted all buckets."
    fi
}

# run_parallel: Executes commands in parallel based on a template and substitutes.
#   Only prints output (stdout/stderr) if a command errors out (non-zero exit status).
#   The function returns a non-zero exit status if any of the parallel commands fail.
#
# Usage: run_parallel "command_template_with_@" "substitute1" "substitute2" ...
#   The '@' in the command_template will be replaced by each substitute argument.
#
# Example:
#   run_parallel "echo 'Processing @' && sleep 1" "itemA" "itemB" "itemC"
#   run_parallel "ping -c 1 @" "localhost" "nonexistent.domain" "google.com"
#   run_parallel "if [ '@' -eq 0 ]; then echo 'Success: 0'; else exit 1; fi" "0" "1" "0" "2"

function run_parallel() {
  local cmd_template="$1"
  shift

  local tmp_base_dir
  # Create a unique temporary directory for this run
  tmp_base_dir=$(mktemp -d) || { log_error "Could not create temporary directory."; return 1; }
  # Ensure the temporary directory is removed on script exit or interrupt
  trap "rm -rf '$tmp_base_dir'" EXIT

  local pids=()
  local -A cmd_info # Associative array: PID -> "output"

  # Launch all commands in the background
  for arg in "$@"; do
    local full_cmd="${cmd_template//@/$arg}"
    local output_file=$(mktemp "${tmp_base_dir}/output.XXXXXX") || { log_error "Could not create temporary output file."; rm -rf "$tmp_base_dir"; return 1; }
    log_info "Queuing Command: $full_cmd"
    ( eval "$full_cmd" > "$output_file" 2>&1 ) &
    local pid=$!
    pids+=("$pid")
    cmd_info["$pid"]="${full_cmd};${output_file}" # Keep pid, full_cmd and output_file in associative array
  done

  local overall_exit_code=0

  # Wait for each background job to finish and process its output
  for pid in "${pids[@]}"; do
    local -a cmd_info_parts
    # Split the stored info string into an array
    IFS=';' read -r -a cmd_info_parts <<< "${cmd_info["$pid"]}"
    local full_cmd="${cmd_info_parts[0]}"
    local output_file="${cmd_info_parts[1]}"
    wait "$pid"
    local command_status=$?
    if [[ "$command_status" -ne 0 ]]; then
      acquire_lock "$LOG_LOCK_FILE"
      echo ""
      echo ""
      echo "--- Parallel Run Failed ---"
      echo "Command: $full_cmd"
      echo "Exit Status: $command_status"
      echo "--- Output of the Command ---:"
      cat "$output_file"
      echo ""
      echo ""
      release_lock "$LOG_LOCK_FILE"
      overall_exit_code=1 # Set overall exit code to non-zero if any command failed
    else
      log_info "Parallel Run Successful: $full_cmd"
    fi
    # Clean up temporary files for the processed command
    rm -f "$output_file"
    # Remove the entry from the associative array
    unset 'cmd_info["$pid"]'
  done

  return "$overall_exit_code"
}

# run_sequential: Executes commands in sequence based on a template and substitutes.
#   Only prints output (stdout/stderr) if a command errors out (non-zero exit status).
#   The function returns a non-zero exit status if any of the sequential commands fail.
#
# Usage: run_sequential "command_template_with_@" "substitute1" "substitute2" ...
#   The '@' in the command_template will be replaced by each substitute argument.
#
# Example:
#   run_sequential "echo 'Processing @' && sleep 1" "itemA" "itemB" "itemC"
#   run_sequential "ping -c 1 @" "localhost" "nonexistent.domain" "google.com"
#   run_sequential "if [ '@' -eq 0 ]; then echo 'Success: 0'; else exit 1; fi" "0" "1" "0" "2"

function run_sequential() {
  local cmd_template="$1"
  shift

  local tmp_base_dir
  # Create a unique temporary directory for this run
  tmp_base_dir=$(mktemp -d) || { log_error "Could not create temporary directory."; return 1; }
  # Ensure the temporary directory is removed on script exit or interrupt
  # Use a function in trap for robustness, especially if tmp_base_dir could be unset or empty
  trap "rm -rf '$tmp_base_dir'" EXIT

  local overall_exit_code=0

  # Execute each command sequentially
  for arg in "$@"; do
    local full_cmd="${cmd_template//@/$arg}"
    local output_file=$(mktemp "${tmp_base_dir}/output.XXXXXX") || { log_error "Could not create temporary output file."; rm -rf "$tmp_base_dir"; return 1; }
    log_info "Queuing Command: $full_cmd"
    # Execute the command and redirect its output to the temporary file
    # Use eval to correctly handle command substitution and complex commands
    eval "$full_cmd" > "$output_file" 2>&1
    local command_status=$?

    if [[ "$command_status" -ne 0 ]]; then
      acquire_lock "$LOG_LOCK_FILE"
      echo ""
      echo ""
      echo "--- Sequential Run Failed ---"
      echo "Command: $full_cmd"
      echo "Exit Status: $command_status"
      echo "--- Output of the Command ---:"
      cat "$output_file"
      echo ""
      echo ""
      release_lock "$LOG_LOCK_FILE"
      overall_exit_code=1 # Set overall exit code to non-zero if any command failed
    else
      log_info "Sequential Run Successful: $full_cmd"
    fi

    # Clean up temporary file for the processed command
    rm -f "$output_file"
  done
  return "$overall_exit_code"
}

# shellcheck disable=SC2317
function test_package() {
  local package_name="$1"
  local bucket_type="$2"
  local bucket_name
  bucket_name=$(create_bucket "$bucket_type")
  # Go Test flags
  
  GO_TEST_CMD_PARTS=(
    "GODEBUG=asyncpreemptoff=1"
    "go"
    "test"
    "-v"
    "-timeout=${INTEGRATION_TEST_TIMEOUT_IN_MINS}m"
    "${INTEGRATION_TEST_PACKAGE_DIR}/${package_name}"
  )
  if [[ "$SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE" == "true" ]]; then
    GO_TEST_CMD_PARTS+=("-short")
  fi
  if [[ "$package_name" == "benchmarking" ]]; then
    GO_TEST_CMD_PARTS+=("-bench=.")
    GO_TEST_CMD_PARTS+=("-benchtime=100x")
  fi
  # Test Binary flags after this.
  GO_TEST_CMD_PARTS+=(
    "-args"
    "-integrationTest"
    "-testbucket=${bucket_name}"
    "-presubmit=${RUN_TESTS_WITH_PRESUBMIT_FLAG}"
    "-testInstalledPackage=${TEST_INSTALLED_PACKAGE}"
  )

  if [[ "$bucket_type" == "zonal" ]]; then
    GO_TEST_CMD_PARTS+=("-zonal")
  fi

  # Use printf %q to quote each argument safely for eval
  # This ensures spaces and special characters within arguments are handled correctly.
  GO_TEST_CMD=$(printf "%q " "${GO_TEST_CMD_PARTS[@]}")

  if ! eval "$GO_TEST_CMD"; then 
    return 1
  else
    return 0
  fi
}

function test_package_hns() {
  test_package "$1" "hns"
}

function test_package_flat() {
  test_package "$1" "flat"
}

function test_package_zonal() {
  test_package "$1" "zonal"
}

function upgrade_gcloud_version() {
  sudo apt-get update
  # Upgrade gcloud version.
  # Kokoro machine's outdated gcloud version prevents the use of the "managed-folders" feature.
  gcloud version
  wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
  sudo tar xzf gcloud.tar.gz && sudo cp -r google-cloud-sdk /usr/local && sudo rm -r google-cloud-sdk
  sudo /usr/local/google-cloud-sdk/install.sh
  gcloud version && rm gcloud.tar.gz
  sudo /usr/local/google-cloud-sdk/bin/gcloud components update
  sudo /usr/local/google-cloud-sdk/bin/gcloud components install alpha
}

function install_packages() {
  # e.g. architecture=arm64 or amd64
  architecture=$(dpkg --print-architecture)
  log_info "Installing go-lang version: ${GO_VERSION}"
  wget -O go_tar.tar.gz https://go.dev/dl/${GO_VERSION}.linux-${architecture}.tar.gz -q
  sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
  rm go_tar.tar.gz
  export PATH=$PATH:/usr/local/go/bin
  sudo apt-get install -y python3
  # install python3-setuptools tools.
  sudo apt-get install -y gcc python3-dev python3-setuptools
  # Downloading composite object requires integrity checking with CRC32c in gsutil.
  # it requires to install crcmod.
  sudo apt install -y python3-crcmod
}

function run_e2e_tests_for_flat_bucket() {
  log_info "Started running e2e tests for flat bucket"
  run_parallel "test_package_flat @" "${PARALLEL_TEST_PACKAGES[@]}" &
  parallel_tests_flat_group_pid=$!

  run_sequential "test_package_flat @" "${SEQUENTIAL_TEST_PACKAGES[@]}" &
  non_parallel_tests_flat_group_pid=$!

  # Wait for all tests to complete.
  wait $parallel_tests_flat_group_pid
  parallel_tests_flat_group_exit_code=$?
  wait $non_parallel_tests_flat_group_pid
  non_parallel_tests_flat_group_exit_code=$?

  if [ $parallel_tests_flat_group_exit_code != 0 ] || [ $non_parallel_tests_flat_group_exit_code != 0 ]; then
    return 1
  fi
  return 0
}

function run_e2e_tests_for_hns_bucket(){
  log_info "Started running e2e tests for HNS bucket"
  run_parallel "test_package_hns @" "${PARALLEL_TEST_PACKAGES[@]}" &
  parallel_tests_hns_group_pid=$!
  run_sequential "test_package_hns @" "${SEQUENTIAL_TEST_PACKAGES[@]}" &
  non_parallel_tests_hns_group_pid=$!

  # Wait for all tests to complete.
  wait $parallel_tests_hns_group_pid
  parallel_tests_hns_group_exit_code=$?
  wait $non_parallel_tests_hns_group_pid
  non_parallel_tests_hns_group_exit_code=$?

  if [ $parallel_tests_hns_group_exit_code != 0 ] || [ $non_parallel_tests_hns_group_exit_code != 0 ]; then
    return 1
  fi
  return 0
}

function run_e2e_tests_for_zonal_bucket(){
  log_info "Started running e2e tests for ZONAL bucket"
  run_parallel "test_package_zonal @" "${PARALLEL_TEST_PACKAGES_FOR_ZB[@]}" &
  parallel_tests_zonal_group_pid=$!
  run_sequential "test_package_zonal @" "${SEQUENTIAL_TEST_PACKAGES_FOR_ZB[@]}" &
  non_parallel_tests_zonal_group_pid=$!

  # Wait for all tests to complete.
  wait $parallel_tests_zonal_group_pid
  parallel_tests_zonal_group_exit_code=$?
  wait $non_parallel_tests_zonal_group_pid
  non_parallel_tests_zonal_group_exit_code=$?

  if [ $parallel_tests_zonal_group_exit_code != 0 ] || [ $non_parallel_tests_zonal_group_exit_code != 0 ]; then
    return 1
  fi
  return 0
}

function run_e2e_tests_for_tpc() {
  local bucket=$1
  if [ "$bucket" == "" ];
  then
    echo "Bucket name is required"
    return 1
  fi

  # Clean bucket before testing.
  gcloud --verbosity=error storage rm -r gs://"$bucket"/*

  # Run Operations e2e tests in TPC to validate all the functionality.
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/... --testOnTPCEndPoint=$RUN_TEST_ON_TPC_ENDPOINT $GO_TEST_SHORT_FLAG $PRESUBMIT_RUN_FLAG --zonal=false -p 1 --integrationTest -v --testbucket="$bucket" --testInstalledPackage=$RUN_E2E_TESTS_ON_PACKAGE -timeout $INTEGRATION_TEST_TIMEOUT
  exit_code=$?

  set -e

  # Delete data after testing.
  gcloud --verbosity=error storage rm -r gs://"$bucket"/*

  if [ $exit_code != 0 ]; then
    return 1
  fi
  return 0
}

function run_e2e_tests_for_emulator() {
  ./tools/integration_tests/emulator_tests/emulator_tests.sh $RUN_E2E_TESTS_ON_PACKAGE
}

function main(){
  # Delete buckets in parallel if program exits.
  trap 'delete_buckets "${BUCKET_NAMES[@]}"' EXIT

  log_info ""
  log_info "------ Started Running e2e Tests ------"
  log_info ""

  set -e

  upgrade_gcloud_version

  install_packages

  set +e

  #run integration tests
  exit_code=0

  if [[ "${RUN_TESTS_WITH_ZONAL_BUCKET}" == "true" ]]; then
    run_e2e_tests_for_zonal_bucket &
    e2e_tests_zonal_bucket_pid=$!
    wait $e2e_tests_zonal_bucket_pid
    e2e_tests_zonal_bucket_status=$?

    if [[ $e2e_tests_zonal_bucket_status != 0 ]]; then
      log_error "The e2e tests for zonal bucket failed."
      exit_code=1
    fi
  else
    # Run tpc test and exit in case RUN_TEST_ON_TPC_ENDPOINT is true.
    if [[ "${RUN_TEST_ON_TPC_ENDPOINT}" == "true" ]]; then
        # Run tests for flat bucket
        run_e2e_tests_for_tpc gcsfuse-e2e-tests-tpc &
        e2e_tests_tpc_flat_bucket_pid=$!
        # Run tests for hns bucket
        run_e2e_tests_for_tpc gcsfuse-e2e-tests-tpc-hns &
        e2e_tests_tpc_hns_bucket_pid=$!

        wait $e2e_tests_tpc_flat_bucket_pid
        e2e_tests_tpc_flat_bucket_status=$?

        wait $e2e_tests_tpc_hns_bucket_pid
        e2e_tests_tpc_hns_bucket_status=$?

        if [[ $e2e_tests_tpc_flat_bucket_status != 0 ]]; then
            log_error "The e2e tests for flat bucket failed."
            exit 1
        fi
        if [[ $e2e_tests_tpc_hns_bucket_status != 0 ]]; then
            log_error "The e2e tests for hns bucket failed."
            exit 1
        fi
        # Exit to prevent the following code from executing for TPC.
        exit 0
    fi

    run_e2e_tests_for_hns_bucket &
    e2e_tests_hns_bucket_pid=$!

    run_e2e_tests_for_flat_bucket &
    e2e_tests_flat_bucket_pid=$!

    run_e2e_tests_for_emulator &
    e2e_tests_emulator_pid=$!

    wait $e2e_tests_emulator_pid
    e2e_tests_emulator_status=$?

    wait $e2e_tests_flat_bucket_pid
    e2e_tests_flat_bucket_status=$?

    wait $e2e_tests_hns_bucket_pid
    e2e_tests_hns_bucket_status=$?

    if [[ $e2e_tests_flat_bucket_status != 0 ]]; then
      log_error "The e2e tests for flat bucket failed."
      exit_code=1
    fi

    if [ $e2e_tests_hns_bucket_status != 0 ]; then
      log_error "The e2e tests for hns bucket failed."
      exit_code=1
    fi

    if [ $e2e_tests_emulator_status != 0 ]; then
      log_error "The e2e tests for emulator failed."
      exit_code=1
    fi
  fi
  exit $exit_code
}

#Main method to run script
main
