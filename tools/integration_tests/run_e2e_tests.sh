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

# Constants
readonly GO_VERSION="go1.24.0"
readonly PROJECT_ID="gcs-fuse-test-ml"
readonly BUCKET_PREFIX="gcs-fuse-e2e-tests"
readonly INTEGRATION_TEST_PACKAGE_DIR="./tools/integration_tests"
readonly INTEGRATION_TEST_TIMEOUT_IN_MINS=90

readonly FIXED_RANDOM_PREFIX=$(head /dev/urandom | tr -dc 'a-z0-9' | head -c 8)
readonly LOG_LOCK_FILE=$(mktemp "/tmp/${FIXED_RANDOM_PREFIX}_logging_lock.XXXXXX")
readonly BUCKET_CREATION_LOCK_FILE=$(mktemp "/tmp/${FIXED_RANDOM_PREFIX}_bucket_creation_lock.XXXXXX")
readonly PACKAGE_STATS_FILE=$(mktemp "/tmp/${FIXED_RANDOM_PREFIX}package_stats.XXXXXX")

# Default values for optional arguments
RUN_TEST_ON_TPC_ENDPOINT=false
RUN_TESTS_WITH_PRESUBMIT_FLAG=false
RUN_TESTS_WITH_ZONAL_BUCKET=false

# Usage Documentation
usage() {
  echo "Usage: $0 <TEST_INSTALLED_PACKAGE> <SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE> <BUCKET_LOCATION> [RUN_TESTS_WITH_PRESUBMIT_FLAG] [RUN_TEST_ON_TPC_ENDPOINT] [RUN_TESTS_WITH_ZONAL_BUCKET]"
  echo "  TEST_INSTALLED_PACKAGE: 'true' or 'false' to test installed gcsfuse package."
  echo "  SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE: 'true' or 'false' to skip few non-essential tests inside packages."
  echo "  BUCKET_LOCATION: The Google Cloud Storage bucket location (e.g., 'us-central1')."
  echo "  RUN_TESTS_WITH_PRESUBMIT_FLAG (optional): 'true' or 'false' to run tests with presubmit flag."
  echo "  RUN_TEST_ON_TPC_ENDPOINT (optional): 'true' or 'false' to run tests on TPC endpoint."
  echo "  RUN_TESTS_WITH_ZONAL_BUCKET (optional): 'true' or 'false' to run tests with zonal bucket."
  exit 1
}

# Logging Helpers
log_info() {
  echo "[INFO] $(date +"%Y-%m-%d %H:%M:%S"): $1"
}

log_error() {
  echo "[ERROR] $(date +"%Y-%m-%d %H:%M:%S"): $1"
}

# Argument Parsing and Assignments
if [ "$#" -lt 3 ]; then
  log_error "Missing required arguments."
  usage
fi
if [ "$#" -gt 6 ]; then
  log_error "Too many arguments."
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
  "stale_handle"
)

# These packages which can only be run in sequential.
SEQUENTIAL_TEST_PACKAGES=(
  "readonly"
)

# Tests Packages which can be run in parallel on zonal buckets.
PARALLEL_TEST_PACKAGES_FOR_ZB=(
  "stale_handle"
)

# These packages which can only be run in sequential on zonal buckets.
SEQUENTIAL_TEST_PACKAGES_FOR_ZB=(
  "readonly"
)

# acquire_lock: Acquires exclusive lock or exits script on failure.
# Args: $1 = path to lock file.
acquire_lock() {
  if [[ -z "$1" ]]; then
    log_error "acquire_lock: Lock file path is required."
    exit 1
  fi
  local lock_file="$1"
  exec 200>"$lock_file" || {
    log_error "Could not open lock file $lock_file."
    exit 1
  }
  flock -x 200 || {
    log_error "Failed to acquire lock on $lock_file."
    exit 1
  }
  return 0
}

# release_lock: Releases lock or exits script on failure.
# Args: $1 = path to lock file
release_lock() {
  if [[ -z "$1" ]]; then
    log_error "release_lock: Lock file path is required."
    exit 1
  fi
  local lock_file="$1"
  [[ -e "/proc/self/fd/200" || -L "/proc/self/fd/200" ]] && exec 200>&- || {
    log_error "Lock file descriptor (FD 200) not open for $lock_file. Possible previous error or double release."
    exit 1
  } # FD not open or close failed
  return 0
}

log_info_locked() {
  acquire_lock "$LOG_LOCK_FILE"
  log_info "$1"
  release_lock "$LOG_LOCK_FILE"
}

log_error_locked() {
  acquire_lock "$LOG_LOCK_FILE"
  log_error "$1"
  release_lock "$LOG_LOCK_FILE"
}

# Stores the names of buckets created using create_bucket.
BUCKET_NAMES=()

# shellcheck disable=SC2317
create_bucket() {
  if [[ $# -ne 1 ]]; then
    log_error_locked "create_bucket called with incorrect number of arguments."
    return 1
  fi
  local bucket_type="$1"
  local uuid
  uuid=$(uuidgen)
  if [[ -z "$uuid" ]]; then
    log_error_locked "Unable to generate random UUID for bucket name"
    return 1
  fi
  local bucket_name="${BUCKET_PREFIX}-${bucket_type}-${uuid}"
  local cmd
  if [[ "$bucket_type" == "flat" ]]; then
    cmd="gcloud alpha storage buckets create gs://${bucket_name} --project=${PROJECT_ID} --location=${BUCKET_LOCATION} --uniform-bucket-level-access > /dev/null 2>&1"
  elif [[ "$bucket_type" == "hns" ]]; then
    cmd="gcloud alpha storage buckets create gs://${bucket_name} --project=${PROJECT_ID} --location=${BUCKET_LOCATION} --uniform-bucket-level-access --enable-hierarchical-namespace > /dev/null 2>&1"
  elif [[ "$bucket_type" == "zonal" ]]; then
    cmd="gcloud alpha storage buckets create gs://${bucket_name} --project=${PROJECT_ID} --location=${BUCKET_LOCATION} --placement=${BUCKET_LOCATION}-a --default-storage-class=RAPID --uniform-bucket-level-access --enable-hierarchical-namespace > /dev/null 2>&1"
  else
    log_error_locked "Invalid Bucket Type [${bucket_type}]. Supported Types [flat, hns, zonal]"
    return 1
  fi
  acquire_lock "$BUCKET_CREATION_LOCK_FILE"
  if ! eval "$cmd"; then
    log_error_locked "Unable to create bucket [${bucket_name}]"
    release_lock "$BUCKET_CREATION_LOCK_FILE"
    return 1
  fi
  sleep 2 # Ensure 2 second gap between creating a new bucket.
  release_lock "$BUCKET_CREATION_LOCK_FILE"
  BUCKET_NAMES+=("$bucket_name")
  echo "$bucket_name"
  return 0
}

# shellcheck disable=SC2317
delete_bucket() {
  if [[ $# -ne 1 ]]; then
    log_error_locked "delete_bucket called with incorrect number of arguments."
    return 1
  fi
  local bucket="$1"
  if ! gcloud -q storage rm -r "gs://${bucket}"; then
    log_error_locked "Unable to delete bucket [${bucket}]"
    return 1
  fi
  return 0
}

# shellcheck disable=SC2317
clean_up() {
  # Clean up temp files.
  if ! rm -rf "tmp/${FIXED_RANDOM_PREFIX}_*"; then 
    log_error_locked "Failed to temporary files"
  else 
    log_info_locked "Successfully cleaned up temporary files"
  fi
  if [[ $# -eq 0 ]]; then
    return 0 # No buckets to delete
  fi
  local buckets=("$@")
  if ! run_parallel "delete_bucket @" "${buckets[@]}"; then
    log_error_locked "Failed to delete all buckets"
  else
    log_info_locked "Successfully deleted all buckets."
  fi
}

# run_parallel: Executes commands in parallel based on a template and substitutes.
#   Prints output (stdout/stderr) if the command errors out.
#   Prints success message if command succeeds.
#   The function returns a non-zero exit status if any of the parallel commands fail.
#
# Usage: run_parallel "command_template_with_@" "substitute1" "substitute2" ...
#   The '@' in the command_template will be replaced by each substitute argument.
#
# Example:
#   run_parallel "echo 'Processing @' && sleep 1" "itemA" "itemB" "itemC"

run_parallel() {
  if [[ $# -lt 2 ]]; then
    log_error_locked "run_parallel called with incorrect number of arguments."
    return 1
  fi
  local cmd_template="$1"
  shift

  local pids=()
  local -A cmd_info # Associative array: PID -> "output"

  # Launch all commands in the background
  for arg in "$@"; do
    local full_cmd="${cmd_template//@/$arg}"
    local output_file=$(mktemp "tmp/${FIXED_RANDOM_PREFIX}_${arg}_output.XXXXXX") || {
      log_error_locked "Could not create temporary output file."
      return 1
    }
    log_info_locked "Queuing Parallel Command: $full_cmd"
    (eval "$full_cmd" >"$output_file" 2>&1) &
    local pid=$!
    pids+=("$pid")
    cmd_info["$pid"]="${full_cmd};${output_file}" # Keep pid, full_cmd and output_file in associative array
  done

  local overall_exit_code=0

  # Wait for each background job to finish and process its output
  for pid in "${pids[@]}"; do
    local -a cmd_info_parts
    # Split the stored info string into an array
    IFS=';' read -r -a cmd_info_parts <<<"${cmd_info["$pid"]}"
    local full_cmd="${cmd_info_parts[0]}"
    local output_file="${cmd_info_parts[1]}"
    wait "$pid"
    local command_status=$?
    if [[ "$command_status" -ne 0 ]]; then
      acquire_lock "$LOG_LOCK_FILE"
      log_error ""
      log_error ""
      log_error "--- Parallel Command Failed ---"
      log_error "Command: $full_cmd"
      log_error "--- Stdout/Stderr ---:"
      cat "$output_file"
      log_error ""
      log_error ""
      release_lock "$LOG_LOCK_FILE"
      overall_exit_code=1 # Set overall exit code to non-zero if any command failed
    else
      log_info_locked "Parallel Command Successful: $full_cmd"
    fi
    # Remove the entry from the associative array
    unset 'cmd_info["$pid"]'
  done

  return "$overall_exit_code"
}

# run_sequential: Executes commands in sequence based on a template and substitutes.
#   Prints output (stdout/stderr) if the command errors out.
#   Prints success message if command succeeds.
#   The function returns a non-zero exit status if any of the sequential commands fail.
#
# Usage: run_sequential "command_template_with_@" "substitute1" "substitute2" ...
#   The '@' in the command_template will be replaced by each substitute argument.
#
# Example:
#   run_sequential "echo 'Processing @' && sleep 1" "itemA" "itemB" "itemC"

run_sequential() {
  if [[ $# -lt 2 ]]; then
    log_error_locked "run_sequential called with incorrect number of arguments."
    return 1
  fi
  local cmd_template="$1"
  shift

  local overall_exit_code=0

  # Execute each command sequentially
  for arg in "$@"; do
    local full_cmd="${cmd_template//@/$arg}"
    local output_file=$(mktemp "tmp/${FIXED_RANDOM_PREFIX}_${arg}_output.XXXXXX") || {
      log_error_locked "Could not create temporary output file."
      return 1
    }
    log_info_locked "Queuing Sequential Command: $full_cmd"
    # Execute the command and redirect its output to the temporary file
    # Use eval to correctly handle command substitution and complex commands
    eval "$full_cmd" >"$output_file" 2>&1
    local command_status=$?

    if [[ "$command_status" -ne 0 ]]; then
      acquire_lock "$LOG_LOCK_FILE"
      log_error ""
      log_error ""
      log_error "--- Sequential Command Failed ---"
      log_error "Command: $full_cmd"
      log_error "--- Stdout/Stderr ---:"
      cat "$output_file"
      log_error ""
      log_error ""
      release_lock "$LOG_LOCK_FILE"
      overall_exit_code=1 # Set overall exit code to non-zero if any command failed
    else
      log_info_locked "Sequential Command Successful: $full_cmd"
    fi

  done
  return "$overall_exit_code"
}

# shellcheck disable=SC2317
test_package() {
  local package_name="$1"
  local bucket_type="$2"
  local bucket_name
  bucket_name=$(create_bucket "$bucket_type")
  if [[ -z "$bucket_name" ]]; then
    log_error_locked "Failed to create bucket of type $bucket_type, name $bucket_name, exit_code $?"
    return 1
  fi
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
  )

  if [[ "$TEST_INSTALLED_PACKAGE" == "true" ]]; then
    GO_TEST_CMD_PARTS+=("-testInstalledPackage")
  fi

  if [[ "$RUN_TESTS_WITH_PRESUBMIT_FLAG" == "true" ]]; then
    GO_TEST_CMD_PARTS+=("-presubmit")
  fi

  if [[ "$bucket_type" == "zonal" ]]; then
    GO_TEST_CMD_PARTS+=("-zonal")
  fi

  if [[ "$RUN_TEST_ON_TPC_ENDPOINT" == "true" ]]; then
    GO_TEST_CMD_PARTS+=("-testOnTPCEndPoint")
  fi

  # Use printf %q to quote each argument safely for eval
  # This ensures spaces and special characters within arguments are handled correctly.
  GO_TEST_CMD=$(printf "%q " "${GO_TEST_CMD_PARTS[@]}")
  local start=$SECONDS
  local exit_code=0
  eval "$GO_TEST_CMD"
  exit_code=$?
  local end=$SECONDS
  # Record stats
  wait_reps=$((start / 60))
  run_reps=$(((end - start + 60) / 60))
  # Build the WWW and RRRR strings
  wait_string=""
  for ((i = 0; i < wait_reps; i++)); do
    wait_string+=" "
  done

  run_string=""
  for ((i = 0; i < run_reps; i++)); do
    run_string+=">"
  done

  exit_status="PASS"
  if [[ "$exit_code" -ne 0 ]]; then
    exit_status="FAIL"
  fi
  combined_time_bar="${wait_string}${run_string}"
  current_package_stats=$(printf "| %-25s | %-15s | %-10s | %-50s|\n" \
    "$package_name" \
    "$bucket_type" \
    "$exit_status" \
    "$combined_time_bar")
  echo "$current_package_stats" >>"$PACKAGE_STATS_FILE"
  if [[ "$exit_code" -ne 0 ]]; then
    return 1
  else
    return 0
  fi
}

print_package_stats() {
  # Sorts package stats by package name and bucket type
  sort -o "$PACKAGE_STATS_FILE" "$PACKAGE_STATS_FILE"
  segment1_hyphens=$(printf '%.s-' {1..27})
  segment2_hyphens=$(printf '%.s-' {1..17})
  segment3_hyphens=$(printf '%.s-' {1..12})
  segment4_hyphens=$(printf '%.s-' {1..51})

  # Concatenate the segments with '+' characters into a single string
  # and then print that string using printf.
  separator=$(printf "+%s+%s+%s+%s+\n" \
    "${segment1_hyphens}" \
    "${segment2_hyphens}" \
    "${segment3_hyphens}" \
    "${segment4_hyphens}")
  echo ""
  echo "Timings for the packages."
  echo "$separator"
  while IFS= read -r line; do
    echo "$line"
    echo "$separator"
  done <"$PACKAGE_STATS_FILE"
  echo ""
}

# shellcheck disable=SC2317
test_package_hns() {
  test_package "$1" "hns"
}

# shellcheck disable=SC2317
test_package_flat() {
  test_package "$1" "flat"
}

# shellcheck disable=SC2317
test_package_zonal() {
  test_package "$1" "zonal"
}

upgrade_gcloud_version() {
  sudo apt-get update
  # Kokoro machine's outdated gcloud version prevents the use of the "managed-folders" feature.
  log_info_locked "Existing Gcloud version."
  gcloud version

  # Download gcloud.tar.gz to the temporary directory
  wget -O "/tmp/${FIXED_RANDOM_PREFIX}_gcloud.tar.gz" https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q

  # Extract gcloud.tar.gz within the temporary directory
  sudo tar xzf "/tmp/${FIXED_RANDOM_PREFIX}_gcloud.tar.gz" -C /tmp/google-cloud-sdk

  # Copy the extracted google-cloud-sdk from the temporary directory to /usr/local
  sudo mv -r /tmp/google-cloud-sdk /usr/local

  # Install gcloud
  sudo /usr/local/google-cloud-sdk/install.sh --quiet
  log_info_locked "Updated Gcloud version."
  # Verify updated gcloud version
  gcloud version

  # Update gcloud components
  sudo /usr/local/google-cloud-sdk/bin/gcloud components update

  # Install alpha components
  sudo /usr/local/google-cloud-sdk/bin/gcloud components install alpha
}

install_packages() {
  # e.g. architecture=arm64 or amd64
  architecture=$(dpkg --print-architecture)
  log_info_locked "Installing go-lang version: ${GO_VERSION}"
  wget -O "/tmp/${FIXED_RANDOM_PREFIX}_go_tar.tar.gz" https://go.dev/dl/${GO_VERSION}.linux-${architecture}.tar.gz -q
  sudo rm -rf /usr/local/go && tar -xzf "tmp/${FIXED_RANDOM_PREFIX}_go_tar.tar.gz" -C /tmp/go && sudo mv /tmp/go /usr/local
  export PATH=$PATH:/usr/local/go/bin
  sudo apt-get install -y python3
  # install python3-setuptools tools.
  sudo apt-get install -y gcc python3-dev python3-setuptools
  # Downloading composite object requires integrity checking with CRC32c in gsutil.
  # it requires to install crcmod.
  sudo apt install -y python3-crcmod
  exit 1
}

run_e2e_tests_for_flat_bucket() {
  log_info_locked "Started running e2e tests for flat bucket"
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
    log_error_locked "The e2e tests for flat bucket failed."
    return 1
  fi
  log_info_locked "The e2e tests for flat bucket successful."
  return 0
}

run_e2e_tests_for_hns_bucket() {
  log_info_locked "Started running e2e tests for HNS bucket"
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
    log_error_locked "The e2e tests for hns bucket failed."
    return 1
  fi
  log_info_locked "The e2e tests for hns bucket successful."
  return 0
}

run_e2e_tests_for_zonal_bucket() {
  log_info_locked "Started running e2e tests for ZONAL bucket"
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
    log_error_locked "The e2e tests for zonal bucket failed."
    return 1
  fi
  log_info_locked "The e2e tests for zonal bucket successful."
  return 0
}

run_e2e_tests_for_tpc() {
  local bucket=$1
  if [[ "$bucket" == "" ]]; then
    log_error_locked "Bucket name is required"
    return 1
  fi
  log_info_locked "Started running e2e tests for tpc for bucket $bucket"
  local tpc_test_log
  emulator_test_log=$(mktemp /tmp/tpc_test_log.XXXXXX)
  trap 'rm "$tpc_test_log"' EXIT
  # Clean bucket before testing.
  gcloud --verbosity=error storage rm -r gs://"$bucket"/* >"$tpc_test_log" 2>&1

  # Run Operations e2e tests in TPC to validate all the functionality.
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/... --testOnTPCEndPoint="$RUN_TEST_ON_TPC_ENDPOINT" "$GO_TEST_SHORT_FLAG" "$PRESUBMIT_RUN_FLAG" --integrationTest -v --testbucket="$bucket" --testInstalledPackage="$RUN_E2E_TESTS_ON_PACKAGE" -timeout "$INTEGRATION_TEST_TIMEOUT" >"$tpc_test_log" 2>&1

  # Delete data after testing.
  gcloud --verbosity=error storage rm -r gs://"$bucket"/* >"$tpc_test_log" 2>&1

  exit_code=$?

  if [[ $exit_code != 0 ]]; then
    acquire_lock "$LOG_LOCK_FILE"
    log_error ""
    log_error ""
    log_error "--- TPC Run Failed for bucket $bucket ---"
    log_error "--- Stdout/Stderr ---"
    cat "$tpc_test_log"
    log_error ""
    log_error ""
    release_lock "$LOG_LOCK_FILE"
    return 1
  fi
  log_info_locked "TPC tests for bucket ${bucket} successful."
  return 0
}

run_e2e_tests_for_emulator() {
  log_info_locked "Started running e2e tests for emulator."
  local emulator_test_log
  emulator_test_log=$(mktemp /tmp/emulator_test_log.XXXXXX)
  trap 'rm "$emulator_test_log"' EXIT

  if ! ./tools/integration_tests/emulator_tests/emulator_tests.sh "$TEST_INSTALLED_PACKAGE" >"$emulator_test_log" 2>&1; then
    acquire_lock "$LOG_LOCK_FILE"
    log_error ""
    log_error ""
    log_error "--- Emulator Run Failed ---"
    log_error "Command: $full_cmd"
    log_error "--- Stdout/Stderr ---:"
    cat "$emulator_test_log"
    log_error ""
    log_error ""
    release_lock "$LOG_LOCK_FILE"
    return 1
  fi
  log_info_locked "Emulator tests successful."
  return 0
}

main() {
  # Clean up everything on exit.
  trap 'clean_up "${BUCKET_NAMES[@]}"' EXIT

  log_info_locked ""
  log_info_locked "------ Upgrading gcloud and installing packages ------"
  log_info_locked ""

  set -e

  upgrade_gcloud_version
  install_packages
  set +e

  log_info_locked "------ Upgrading gcloud and installing packages took $SECONDS seconds ------"

  log_info_locked ""
  log_info_locked "------ Started running E2E test packages ------"
  log_info_locked ""
  # Reset SECONDS to 0
  SECONDS=0
  # Set exit code of e2e run to 0
  exit_code=0
  if [[ "${RUN_TESTS_WITH_ZONAL_BUCKET}" == "true" ]]; then
    run_e2e_tests_for_zonal_bucket &
    e2e_tests_zonal_bucket_pid=$!
    wait $e2e_tests_zonal_bucket_pid
    exit_code=$((exit_code || $? != 0))
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
      exit_code=$((exit_code || $? != 0))

      wait $e2e_tests_tpc_hns_bucket_pid
      exit_code=$((exit_code || $? != 0))
    else
      run_e2e_tests_for_hns_bucket &
      e2e_tests_hns_bucket_pid=$!

      run_e2e_tests_for_flat_bucket &
      e2e_tests_flat_bucket_pid=$!

      run_e2e_tests_for_emulator &
      e2e_tests_emulator_pid=$!

      wait $e2e_tests_emulator_pid
      exit_code=$((exit_code || $? != 0))

      wait $e2e_tests_flat_bucket_pid
      exit_code=$((exit_code || $? != 0))

      wait $e2e_tests_hns_bucket_pid
      exit_code=$((exit_code || $? != 0))
    fi
  fi
  print_package_stats
  log_info_locked "------ E2E test packages complete run took $SECONDS seconds ------"
  exit $exit_code
}

#Main method to run script
main
