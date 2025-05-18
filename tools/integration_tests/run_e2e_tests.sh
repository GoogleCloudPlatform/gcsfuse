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
readonly DEFUALT_PROJECT_ID="gcs-fuse-test-ml"
readonly TPCZERO_PROJECT_ID="tpczero-system:gcsfuse-test-project"
readonly TPC_BUCKET_LOCATION="u-us-prp1"
readonly BUCKET_PREFIX="gcsfuse-e2e"
readonly INTEGRATION_TEST_PACKAGE_DIR="./tools/integration_tests"
readonly INTEGRATION_TEST_TIMEOUT_IN_MINS=90
readonly TMP_PREFIX="gcsfuse_e2e"
readonly LOG_LOCK_FILE=$(mktemp "/tmp/${TMP_PREFIX}_logging_lock.XXXXXX")
readonly BUCKET_NAMES=$(mktemp "/tmp/${TMP_PREFIX}_bucket_names.XXXXXX")
readonly PACKAGE_STATS_FILE=$(mktemp "/tmp/${TMP_PREFIX}_package_stats.XXXXXX")
readonly VM_USAGE=$(mktemp "/tmp/${TMP_PREFIX}_vm_usage.XXXXXX")
readonly PARALLELISM=1

# Default values for optional arguments
RUN_TEST_ON_TPC_ENDPOINT=false
RUN_TESTS_WITH_PRESUBMIT_FLAG=false
RUN_TESTS_WITH_ZONAL_BUCKET=false

# Set default project id for tests.
PROJECT_ID="${DEFUALT_PROJECT_ID}"


# Usage Documentation
usage() {
  echo "Usage: $0 <TEST_INSTALLED_PACKAGE> <SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE> <BUCKET_LOCATION> [RUN_TEST_ON_TPC_ENDPOINT] [RUN_TESTS_WITH_PRESUBMIT_FLAG] [RUN_TESTS_WITH_ZONAL_BUCKET]"
  echo "  TEST_INSTALLED_PACKAGE: 'true' or 'false' to test installed gcsfuse package."
  echo "  SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE: 'true' or 'false' to skip few non-essential tests inside packages."
  echo "  BUCKET_LOCATION: The Google Cloud Storage bucket location (e.g., 'us-central1')."
  echo "  RUN_TEST_ON_TPC_ENDPOINT (optional): 'true' or 'false' to run tests on TPC endpoint (Default: 'false')."
  echo "  RUN_TESTS_WITH_PRESUBMIT_FLAG (optional): 'true' or 'false' to run tests with presubmit flag (Default: 'false')."
  echo "  RUN_TESTS_WITH_ZONAL_BUCKET (optional): 'true' or 'false' to run tests with zonal bucket (Default: 'false')."
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
  RUN_TEST_ON_TPC_ENDPOINT="$1"
  shift
fi
if [ -n "$1" ]; then
  RUN_TESTS_WITH_PRESUBMIT_FLAG="$1"
  shift
fi
if [ -n "$1" ]; then
  RUN_TESTS_WITH_ZONAL_BUCKET="$1"
  shift
fi

# Test packages which can be run in parallel.
PARALLEL_TEST_PACKAGES=(
  "managed_folders"
  "operations"
  "concurrent_operations"
  "read_large_files"
  "read_cache"
  "monitoring"
  "local_file"
  "log_rotation"
  "mounting"
  # "grpc_validation"
  "gzip"
  "write_large_files"
  "list_large_dir"
  "rename_dir_limit"
  "explicit_dir"
  "implicit_dir"
  "interrupt"
  "kernel_list_cache"
  "benchmarking"
  "mount_timeout"
  "stale_handle"
  "negative_stat_cache"
  "streaming_writes"
  "readonly"
  "readonly_creds"
)


# Test packages which can be run in parallel on zonal buckets.
PARALLEL_TEST_PACKAGES_FOR_ZB=("${PARALLEL_TEST_PACKAGES[@]}" "unfinalized_object")



# Test packages which can be run in parallel on TPC universe.
PARALLEL_TEST_PACKAGES_FOR_TPC=(
  "operations"
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

# shellcheck disable=SC2317
create_bucket() {
  if [[ $# -ne 2 ]]; then
    log_error "create_bucket() called with incorrect number of arguments."
    return 1
  fi
  local package="$1"
  local bucket_type="$2"
  local bucket_name="${BUCKET_PREFIX}-${package}-${bucket_type}-$(date +%s%N)"
  local bucket_cmd_parts=(
    "gcloud"
    "alpha"
    "storage"
    "buckets"
    "create"
    "gs://${bucket_name}"
    "--project=${PROJECT_ID}"
    "--location=${BUCKET_LOCATION}"
    "--uniform-bucket-level-access"
  )
  if [[ "$bucket_type" == "hns" ]]; then
    bucket_cmd_parts+=("--enable-hierarchical-namespace")
  elif [[ "$bucket_type" == "zonal" ]]; then
    bucket_cmd_parts+=("--enable-hierarchical-namespace")
    bucket_cmd_parts+=("--placement=${BUCKET_LOCATION}-a")
    bucket_cmd_parts+=("--default-storage-class=RAPID")
  elif [[ "$bucket_type" != "flat" ]]; then
    log_error_locked "Invalid bucket type: $bucket_type."
    return 1
  fi
  local bucket_cmd=$(printf "%q " "${bucket_cmd_parts[@]}")
  local bucket_cmd_log=$(mktemp "/tmp/${TMP_PREFIX}_bucket_cmd_log.XXXXXX")
  sleep 4 # Ensure 4 second gap between creating a new bucket.
  if ! eval "$bucket_cmd" > "$bucket_cmd_log" 2>&1; then
    log_error "Unable to create bucket [${bucket_name}]"
    cat "$bucket_cmd_log"
    return 1
  fi
  echo "$bucket_name" >> "$BUCKET_NAMES" # Add bucket names to file.
  echo "$bucket_name"
  return 0
}

setup_package_buckets () {
  if [[ "$#" -ne 3 ]]; then 
    log_error "setup_buckets() called with incorrect number of arguments."
    exit 1
  fi
  local -n package_array="$1"
  local -n package_bucket_array="$2"
  local bucket_type="$3"
  for package in "${package_array[@]}"; do
    local bucket_name
    bucket_name=$(create_bucket "$package" "$bucket_type")
    if [[ $? -eq 0 ]]; then
      package_bucket_array+=("${package} ${bucket_name} ${bucket_type}")
    else
      log_error_locked "Unable to create a bucket for package [${package}] of type [${bucket_type}]."
    fi
  done
}

# shellcheck disable=SC2317
delete_bucket() {
  sleep 5
  if [[ $# -ne 1 ]]; then
    log_error_locked "delete_bucket() called with incorrect number of arguments."
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
  local buckets=()
  # Read each line from BUCKET_NAMES into buckets array
  # This ensures each bucket name is treated as a separate item.
  while IFS= read -r line || [[ -n "$line" ]]; do # Process even if last line has no newline
    buckets+=("$line")
  done < "$BUCKET_NAMES"
  # Clean up buckets if any.
  local clean_up_log=$(mktemp "/tmp/${TMP_PREFIX}_clean_up_log.XXXXXX")
  if [[ "${#buckets[@]}" -gt 0 ]]; then
      if ! run_parallel "$PARALLELISM" "delete_bucket @" "${buckets[@]}" > "$clean_up_log" 2>&1; then
        log_error "Failed to delete all buckets"
      else
        log_info "Successfully deleted all buckets."
    fi
  fi
  if ! rm -rf /tmp/"${TMP_PREFIX}_"*; then 
    log_error "Failed to delete temporary files"
  else 
    log_info "Successfully cleaned up temporary files"
  fi
}

# run_parallel: Executes commands in parallel based on a template and substitutes.
#   Prints output (stdout/stderr) if the command errors out.
#   Prints success message if command succeeds.
#   The function returns a non-zero exit status if any of the parallel commands fail.
#
# Usage: run_parallel "parallelism" "command_template_with_@" "substitute1" "substitute2" ...
#   The '@' in the command_template will be replaced by each substitute argument.
#   This first argument is exten of parallelism for this command.
#
# Example:
#   run_parallel 2 "echo 'Processing @' && sleep 1" "itemA" "itemB" "itemC"
# This command will run at max 2 commands in parallel.

run_parallel() {
  if [[ $# -lt 2 ]]; then
    log_error_locked "run_parallel() called with incorrect number of arguments."
    return 1
  fi
  local parallelism="$1"
  shift
  local cmd_template="$1"
  shift
  local pids=()
  local -A cmd_info # Associative array: PID -> "output"
  local overall_exit_code=0
  process_a_pid() {
    local -n cmd_info_ref="$1"
    local pid
    local status
    wait -n -p pid
    status=$?
    IFS=';' read -r -a cmd_info_parts <<< "${cmd_info_ref["$pid"]}"
    local full_cmd="${cmd_info_parts[0]}"
    local output_file="${cmd_info_parts[1]}"
    # Remove the entry from the associative array
    unset "cmd_info[$pid]"
    if [[ "$status" -ne 0 ]]; then
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
      return 1 # Return 1 to indicate a failure
    else
      log_info_locked "Parallel Command Successful: $full_cmd"
      return 0 # Return 0 to indicate success
    fi
  }
  # Launch commands in the background based on parallelism.
  for arg in "$@"; do
    sleep 120
    local full_cmd="${cmd_template//@/$arg}"
    local output_file=$(mktemp "/tmp/${TMP_PREFIX}_${arg}_output.XXXXXX") || {
      log_error_locked "Could not create temporary output file."
      return 1
    }
    log_info_locked "Executing Parallel Command: $full_cmd"
    (eval "$full_cmd" >"$output_file" 2>&1) &
    local pid=$!
    pids+=("$pid")
    cmd_info["$pid"]="${full_cmd};${output_file}" # Keep pid, full_cmd and output_file in associative array
    if [[ ${#cmd_info[@]} -eq $parallelism ]]; then
      process_a_pid "cmd_info"
      overall_exit_code=$((overall_exit_code || $? != 0))
    fi
  done
  # Process any remaining cmds
  while [ ${#cmd_info[@]} -gt  0 ]
  do
    process_a_pid "cmd_info"
    overall_exit_code=$((overall_exit_code || $? != 0))
  done

  return "$overall_exit_code"
}

# shellcheck disable=SC2317
test_package() {
  if [[ $# -ne 3 ]]; then
    log_error_locked "test_package() called with incorrect number of arguments."
    exit 1
  fi
  local package_name="$1"
  local bucket_name="$2"
  local bucket_type="$3"

  # Build go package test command.
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
  
  # Run the package test command
  local package_status="PASSED"
  local start=$SECONDS
  eval "$GO_TEST_CMD"
  if [[ $? -ne 0 ]]; then
    package_status="FAILED"
  fi
  local end=$SECONDS
  
  # Record stats and build wait run time string.
  # Using each _ char for 1 min wait time and each > char for 1 min run time.
  wait_min=$((start / 60))
  run_min=$(((end - start) / 60))
  current_package_stats=$(printf "| %-25s | %-15s | %-10s |%-60s|\n" \
    "$package_name" \
    "$bucket_type" \
    "$package_status" \
    "$(printf '%0.s_' $(seq 1 "$wait_min"))$(printf '%0.s>' $(seq 1 "$run_min"))") # Produces string like ___>>>
  
  echo "$current_package_stats" >> "$PACKAGE_STATS_FILE"
  if [[ "$package_status" == "FAILED" ]]; then
    return 1
  fi
  return 0
}

print_package_stats() {
  # Sorts package stats by package name and bucket type
  sort -o "$PACKAGE_STATS_FILE" "$PACKAGE_STATS_FILE"
  # separator is a line like +------+----+-----+----+
  separator=$(printf "+%s+%s+%s+%s+\n" \
    "$(printf '%.s-' {1..27})" \
    "$(printf '%.s-' {1..17})" \
    "$(printf '%.s-' {1..12})" \
    "$(printf '%.s-' {1..60})")
  echo ""
  echo "Timings for the e2e test packages run are listed below."
  echo "_ is 1 min wait"
  echo "> is 1 min run"
  echo "$separator"
  printf "| %-25s | %-15s | %-10s | %-25s %s %+25s|\n" \
    "Package Name" "Bucket Type" "Status" "0 min " "runtime" "60 min"
  echo "$separator"
  while IFS= read -r line; do
    echo "$line"
    echo "$separator"
  done <"$PACKAGE_STATS_FILE"
  echo ""
}

upgrade_gcloud_version() {
  sudo apt-get update
  # Upgrade gcloud version.
  # Kokoro machine's outdated gcloud version prevents the use of the "managed-folders" feature.
  gcloud version
  wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
  sudo tar xzf gcloud.tar.gz && sudo cp -r google-cloud-sdk /usr/local && sudo rm -r google-cloud-sdk
  sudo /usr/local/google-cloud-sdk/install.sh -q
  export PATH=/usr/local/google-cloud-sdk/bin:$PATH
  echo 'export PATH=/usr/local/google-cloud-sdk/bin:$PATH' >> ~/.bashrc
  gcloud version && rm gcloud.tar.gz
  sudo /usr/local/google-cloud-sdk/bin/gcloud components update
  sudo /usr/local/google-cloud-sdk/bin/gcloud components install alpha
}

install_packages() {
  # e.g. architecture=arm64 or amd64
  architecture=$(dpkg --print-architecture)
  echo "Installing go-lang 1.24.0..."
  wget -O go_tar.tar.gz https://go.dev/dl/go1.24.0.linux-${architecture}.tar.gz -q
  sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
  rm -rf go_tar.tar.gz
  export PATH=$PATH:/usr/local/go/bin
  sudo apt-get install -y python3
  # install python3-setuptools tools.
  sudo apt-get install -y gcc python3-dev python3-setuptools
  # Downloading composite object requires integrity checking with CRC32c in gsutil.
  # it requires to install crcmod.
  sudo apt install -y python3-crcmod
}

run_e2e_tests_for_flat_bucket() {
  log_info_locked "Started running e2e tests for flat bucket"
  parallel_package_flat_bucket=()
  setup_package_buckets "PARALLEL_TEST_PACKAGES" "parallel_package_flat_bucket" "flat"
  run_parallel "$PARALLELISM" "test_package @" "${parallel_package_flat_bucket[@]}" &
  parallel_tests_flat_group_pid=$!

  # Wait for all tests to complete.
  wait $parallel_tests_flat_group_pid
  parallel_tests_flat_group_exit_code=$?

  if [ $parallel_tests_flat_group_exit_code != 0 ]; then
    log_error_locked "The e2e tests for flat bucket failed."
    return 1
  fi
  log_info_locked "The e2e tests for flat bucket successful."
  return 0
}

run_e2e_tests_for_hns_bucket() {
  log_info_locked "Started running e2e tests for HNS bucket"
  parallel_package_hns_bucket=()
  setup_package_buckets "PARALLEL_TEST_PACKAGES" "parallel_package_hns_bucket" "hns"
  run_parallel "$PARALLELISM" "test_package @" "${parallel_package_hns_bucket[@]}" &
  parallel_tests_hns_group_pid=$!

  # Wait for all tests to complete.
  wait $parallel_tests_hns_group_pid
  parallel_tests_hns_group_exit_code=$?

  if [ $parallel_tests_hns_group_exit_code != 0 ]; then
    log_error_locked "The e2e tests for hns bucket failed."
    return 1
  fi
  log_info_locked "The e2e tests for hns bucket successful."
  return 0
}

run_e2e_tests_for_zonal_bucket() {
  log_info_locked "Started running e2e tests for ZONAL bucket"
  parallel_package_zonal_bucket=()
  setup_package_buckets "PARALLEL_TEST_PACKAGES_FOR_ZB" "parallel_package_zonal_bucket" "zonal"
  run_parallel "$PARALLELISM" "test_package @" "${parallel_package_zonal_bucket[@]}" &
  parallel_tests_zonal_group_pid=$!

  # Wait for all tests to complete.
  wait $parallel_tests_zonal_group_pid
  parallel_tests_zonal_group_exit_code=$?

  if [ $parallel_tests_zonal_group_exit_code != 0 ]; then
    log_error_locked "The e2e tests for zonal bucket failed."
    return 1
  fi
  log_info_locked "The e2e tests for zonal bucket successful."
  return 0
}

run_e2e_tests_for_tpc() {
  log_info_locked "Started running e2e tests for TPC bucket"

  parallel_package_bucket_hns=()
  setup_package_buckets "PARALLEL_TEST_PACKAGES_FOR_TPC" "parallel_package_bucket_hns" "hns"
  run_parallel "$PARALLELISM" "test_package @" "${parallel_package_bucket_hns[@]}" &
  parallel_tests_hns_group_pid=$!

  parallel_package_bucket_flat=()
  setup_package_buckets "PARALLEL_TEST_PACKAGES_FOR_TPC" "parallel_package_bucket_flat" "flat"
  run_parallel "$PARALLELISM" "test_package @" "${parallel_package_bucket_flat[@]}" &
  parallel_tests_flat_group_pid=$!

  # Wait for all tests to complete.
  wait $parallel_tests_hns_group_pid
  parallel_tests_hns_group_exit_code=$?
  wait $parallel_tests_flat_group_pid
  parallel_tests_flat_group_exit_code=$?

  if [ $parallel_tests_hns_group_exit_code != 0 ] || [ $parallel_tests_flat_group_exit_code != 0 ]; then
    log_error_locked "The e2e tests for TPC bucket failed."
    return 1
  fi
  log_info_locked "The e2e tests for TPC bucket successful."
  return 0
}

run_e2e_tests_for_emulator() {
  log_info_locked "Started running e2e tests for emulator."
  local emulator_test_log=$(mktemp "/tmp/${TMP_PREFIX}_emulator_test_log.XXXXXX")
  if ! ./tools/integration_tests/emulator_tests/emulator_tests.sh "$TEST_INSTALLED_PACKAGE" >"$emulator_test_log" 2>&1; then
    acquire_lock "$LOG_LOCK_FILE"
    log_error ""
    log_error ""
    log_error "--- Emulator Run Failed ---"
    log_error "Command: $full_cmd"
    log_error "--- Stdout/Stderr ---"
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
  trap clean_up EXIT SIGINT SIGTERM
  chmod +x ./tools/integration_tests/monitor_vm_usage.sh
  ./tools/integration_tests/monitor_vm_usage.sh "$VM_USAGE" &
  usage_pid=$!
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
      BUCKET_LOCATION="$TPC_BUCKET_LOCATION"
      PROJECT_ID="$TPCZERO_PROJECT_ID"
      run_e2e_tests_for_tpc &
      e2e_tests_tpc_pid=$!
    
      wait $e2e_tests_tpc_pid
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
  log_info_locked "------ E2E test packages complete run took $((SECONDS / 60)) minutes ------"
  log_info_locked ""
  kill -SIGTERM "$usage_pid"
  cat "$VM_USAGE"
  exit $exit_code
}

#Main method to run script
main
