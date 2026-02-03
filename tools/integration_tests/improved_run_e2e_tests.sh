#!/bin/bash
# Copyright 2025 Google LLC
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

# Script Usage Documentation
usage() {
  echo "Usage: $0 --bucket-location <bucket-location> [options]"
  echo "    --bucket-location            <location>      The Google Cloud Storage bucket location (e.g., 'us-central1')."
  echo ""
  echo "Options:"
  echo "    --test-installed-package                     Test installed gcsfuse package. (Default: false)"
  echo "    --skip-non-essential-tests                   Skip non-essential tests inside packages. (Default: false)"
  echo "    --test-on-tpc-endpoint                       Run tests on TPC endpoint. (Default: false)"
  echo "    --presubmit                                  Run tests with presubmit flag. (Default: false)"
  echo "    --zonal                                      Run tests with zonal bucket in --bucket-location region."
  echo "                                                 The placement for Zonal buckets by deafault is Zone A of --bucket-location. (Default: false)"
  echo "    --no-build-binary-in-script                  To disable building gcsfuse binary in script. (Default: false)"
  echo "    --package-level-parallelism   <parallelism>  To adjust the number of packages to execute in parallel. (Default: 10)"
  echo "    --track-resource-usage                       To track resource(cpu/mem/disk) usage during e2e run. (Default: false)"
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

# Confirm bash version before continuing script.
REQUIRED_BASH_MAJOR=5
REQUIRED_BASH_MINOR=1
if (( BASH_VERSINFO[0] < REQUIRED_BASH_MAJOR || ( BASH_VERSINFO[0] == REQUIRED_BASH_MAJOR && BASH_VERSINFO[1] < REQUIRED_BASH_MINOR ) )); then
    log_error "This script requires Bash version: ${REQUIRED_BASH_MAJOR}.${REQUIRED_BASH_MINOR} or higher."
    log_error "You are currently using Bash version: ${BASH_VERSINFO[0]}.${BASH_VERSINFO[1]}"
    exit 1
fi
log_info "Bash version: ${BASH_VERSINFO[0]}.${BASH_VERSINFO[1]}"

# Constants
readonly GO_VERSION=$(cat .go-version)
readonly DEFAULT_PROJECT_ID="gcs-fuse-test-ml"
readonly TPCZERO_PROJECT_ID="tpczero-system:gcsfuse-test-project"
readonly TPC_BUCKET_LOCATION="u-us-prp1"
readonly BUCKET_PREFIX="gcsfuse-e2e"
readonly INTEGRATION_TEST_PACKAGE_DIR="./tools/integration_tests"
readonly INTEGRATION_TEST_PACKAGE_TIMEOUT_IN_MINS=90 
readonly TMP_PREFIX="gcsfuse_e2e"
readonly ZONAL_BUCKET_SUPPORTED_LOCATIONS=("us-central1" "us-west4")
# e2e buckets created are retained for upto 10 days before deletion.
readonly BUCKET_RETENTION_PERIOD_DAYS=10
# 6 second delay between creating buckets as both hns and flat runs create buckets in parallel.
# Ref: https://cloud.google.com/storage/quotas#buckets
readonly DELAY_BETWEEN_BUCKET_CREATION=6
readonly ZONAL="zonal"
readonly FLAT="flat"
readonly HNS="hns"
# Max number of batches of buckets to delete in a single run.
readonly MAX_BUCKET_DELETION_BATCHES=50
# Number of buckets to delete in a single batch.
readonly BUCKET_DELETION_BATCH_SIZE=10

# Set default project id for tests.
PROJECT_ID="${DEFAULT_PROJECT_ID}"
# This variable will store the path if the script builds GCSFuse binaries (gcsfuse, mount.gcsfuse)
BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR=""

LOG_LOCK_FILE=$(mktemp "/tmp/${TMP_PREFIX}_logging_lock.XXXXXX") || { log_error "Unable to create lock file"; exit 1; }
BUCKET_CREATION_LOCK_FILE=$(mktemp "/tmp/${TMP_PREFIX}_bucket_creation_lock.XXXXXX") || { log_error "Unable to create bucket creation lock file"; exit 1; }
PACKAGE_RUNTIME_STATS=$(mktemp "/tmp/${TMP_PREFIX}_package_stats_runtime.XXXXXX") || { log_error "Unable to create package stats runtime file"; exit 1; }
RESOURCE_USAGE_FILE=$(mktemp "/tmp/${TMP_PREFIX}_system_resource_usage.XXXXXX") || { log_error "Unable to create system resource usage file"; exit 1; }

KOKORO_DIR_AVAILABLE=false
if [[ -n "$KOKORO_ARTIFACTS_DIR" ]]; then
  KOKORO_DIR_AVAILABLE=true
fi

# Argument Parsing and Assignments
# Set default values for optional arguments
SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE=false
TEST_INSTALLED_PACKAGE=false
RUN_TEST_ON_TPC_ENDPOINT=false
RUN_TESTS_WITH_PRESUBMIT_FLAG=false
RUN_TESTS_WITH_ZONAL_BUCKET=false
BUILD_BINARY_IN_SCRIPT=true
TRACK_RESOURCE_USAGE=false
PACKAGE_LEVEL_PARALLELISM=10 # Controls how many test packages are run in parallel for hns, flat or zonal buckets.

# Define options for getopt
# A long option name followed by a colon indicates it requires an argument.
LONG=bucket-location:,test-installed-package,skip-non-essential-tests,no-build-binary-in-script,test-on-tpc-endpoint,presubmit,zonal,package-level-parallelism:,track-resource-usage,help

# Parse the options using getopt
# --options "" specifies that there are no short options.
PARSED=$(getopt --options "" --longoptions "$LONG" --name "$0" -- "$@")
if [[ $? -ne 0 ]]; then
    # getopt will have already printed an error message
    usage 1
fi

# Read the parsed options back into the positional parameters.
eval set -- "$PARSED"

# Loop through the options and assign values to our variables
while (( $# >= 1 )); do
    case "$1" in
        --bucket-location)
            BUCKET_LOCATION="$2"
            shift 2
            ;;
        --package-level-parallelism)
            PACKAGE_LEVEL_PARALLELISM="$2"
            shift 2
            ;;
        --test-installed-package)
            TEST_INSTALLED_PACKAGE=true
            shift 
            ;;
        --skip-non-essential-tests)
            SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE=true
            shift
            ;;
        --no-build-binary-in-script)
            BUILD_BINARY_IN_SCRIPT=false
            shift
            ;;
        --test-on-tpc-endpoint)
            RUN_TEST_ON_TPC_ENDPOINT=true
            shift
            ;;
        --presubmit)
            RUN_TESTS_WITH_PRESUBMIT_FLAG=true
            shift
            ;;
        --zonal)
            RUN_TESTS_WITH_ZONAL_BUCKET=true
            shift
            ;;
        --track-resource-usage)
            TRACK_RESOURCE_USAGE=true
            shift
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

# Validates option value to be non-empty and should not be another option name.
validate_option_value() {
  local option=$1
  local value=$2
  if [[ -z "$value" || "$value" == -* ]]; then
    log_error "Invalid or empty value [$value] for option $option."
    usage 1
  fi
}

# Validate long options which require values.
validate_option_value "--bucket-location" "$BUCKET_LOCATION"
validate_option_value "--package-level-parallelism" "$PACKAGE_LEVEL_PARALLELISM"

# Zonal Bucket location validation.
if ${RUN_TESTS_WITH_ZONAL_BUCKET}; then
  supported_bucket=false
  for location in "${ZONAL_BUCKET_SUPPORTED_LOCATIONS[@]}"; do
    if [[ "$BUCKET_LOCATION" == "$location" ]]; then
      supported_bucket=true
      break
    fi
  done
  if ! ${supported_bucket}; then
    log_error "Unsupported Bucket Location ${BUCKET_LOCATION} for Zonal Run. Supported Locations are: ${ZONAL_BUCKET_SUPPORTED_LOCATIONS[*]}"
    exit 1
  fi
fi

# Test packages which can be run for both Zonal and Regional buckets.
# Sorted list descending run times. (Longest Processing Time first strategy) 
TEST_PACKAGES_COMMON=(
  "managed_folders"
  "operations"
  "read_large_files"
  "concurrent_operations"
  # "read_cache"
  "list_large_dir"
  "mount_timeout"
  "write_large_files"
  "implicit_dir"
  "interrupt"
  "local_file"
  "readonly"
  "readonly_creds"
  "rename_dir_limit"
  "kernel_list_cache"
  "streaming_writes"
  "benchmarking"
  "explicit_dir"
  "gzip"
  "log_rotation"
  "monitoring"
  "mounting"
  "unsupported_path"
  # "grpc_validation"
  "negative_stat_cache"
  "stale_handle"
  "release_version"
  "readdirplus"
  "dentry_cache"
  "buffered_read"
  "flag_optimizations"
)

# Test packages for regional buckets.
TEST_PACKAGES_FOR_RB=("${TEST_PACKAGES_COMMON[@]}" "read_cache" "inactive_stream_timeout" "cloud_profiler" "requester_pays_bucket")
# Test packages for zonal buckets.
TEST_PACKAGES_FOR_ZB=("${TEST_PACKAGES_COMMON[@]}" "rapid_appends" "unfinalized_object")
# Test packages for TPC buckets.
TEST_PACKAGES_FOR_TPC=("operations")

# acquire_lock: Acquires exclusive lock or exits script on failure.
# Args: $1 = path to lock file.
acquire_lock() {
  if [[ -z "$1" ]]; then
    log_error "acquire_lock: Lock file path is required."
    exit 1
  fi
  local lock_file="$1"
  local timeout_seconds=600 # 10 minutes
  exec 200>"$lock_file" || {
    log_error "Could not open lock file $lock_file."
    exit 1
  }
  # Attempt to acquire the lock with a timeout
  if ! flock -x -w "$timeout_seconds" 200; then
    log_error "Failed to acquire lock on $lock_file within $timeout_seconds seconds."
    # Close the file descriptor if the lock was not acquired
    exec 200>&-
    exit 1
  fi
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

# logs info to stdout exclusively. used in background commands to ensure logs aren't interleaved.
log_info_locked() {
  acquire_lock "$LOG_LOCK_FILE"
  log_info "$1"
  release_lock "$LOG_LOCK_FILE"
}

# logs error to stdout exclusively. Used in background commands to ensure logs aren't interleaved.
log_error_locked() {
  acquire_lock "$LOG_LOCK_FILE"
  log_error "$1"
  release_lock "$LOG_LOCK_FILE"
}

# Helper method to create "flat", "hns" or "zonal" bucket.
create_bucket() {
  if [[ $# -ne 2 ]]; then
    log_error "create_bucket() called with incorrect number of arguments."
    return 1
  fi
  local package="$1"
  local bucket_type="$2"
  local bucket_name="${BUCKET_PREFIX}-${package}-${bucket_type}-$(date +%s%N)"
  local bucket_cmd_parts=("gcloud" "alpha" "storage" "buckets" "create" "gs://${bucket_name}" "--project=${PROJECT_ID}" "--location=${BUCKET_LOCATION}" "--uniform-bucket-level-access")
  if [[ "$bucket_type" == "$HNS" ]]; then
    bucket_cmd_parts+=("--enable-hierarchical-namespace")
  elif [[ "$bucket_type" == "$ZONAL" ]]; then
    bucket_cmd_parts+=("--enable-hierarchical-namespace" "--placement=${BUCKET_LOCATION}-a" "--default-storage-class=RAPID")
  elif [[ "$bucket_type" != "$FLAT" ]]; then
    log_error "Invalid bucket type: $bucket_type."
    return 1
  fi
  local bucket_cmd bucket_cmd_log attempt=5
  bucket_cmd=$(printf "%q " "${bucket_cmd_parts[@]}")
  bucket_cmd_log=$(mktemp "/tmp/${TMP_PREFIX}_bucket_cmd_log.XXXXXX")
  while : ; do
    attempt=$((attempt - 1))
    if [ $attempt -lt 0 ]; then
      log_error "Unable to create bucket [${bucket_name}] after 5 attempts." 
      cat "$bucket_cmd_log"
      return 1
    fi
    acquire_lock "$BUCKET_CREATION_LOCK_FILE"
    eval "$bucket_cmd" > "$bucket_cmd_log" 2>&1
    local status=$?
    sleep "$DELAY_BETWEEN_BUCKET_CREATION" # have 6 seconds gap between creating buckets.
    release_lock "$BUCKET_CREATION_LOCK_FILE"
    if [ $status -eq 0 ]; then
      break
    fi
  done
  echo "$bucket_name"
  rm -rf "$bucket_cmd_log"
  return 0
}

# Helper method to cleanup expired buckets.
cleanup_expired_buckets() {
    log_info "Deleting buckets older than ${BUCKET_RETENTION_PERIOD_DAYS} days having prefix ${BUCKET_PREFIX}."
    local bucket_list_output
    bucket_list_output=$(gcloud storage buckets list --project=${PROJECT_ID} "gs://${BUCKET_PREFIX}*" \
        --filter="creation_time < -P${BUCKET_RETENTION_PERIOD_DAYS}D" \
        --format="value(storage_url)")
    if [[ $? -ne 0 ]]; then
        log_error "Failed to list expired buckets. Cleanup of expired buckets is skipped."
        return 1
    fi

    if [[ -z "$bucket_list_output" ]]; then
        log_info "No expired buckets found matching the criteria."
        return 0
    fi

    local -a bucket_uris
    mapfile -t bucket_uris <<< "$bucket_list_output"

    log_info "Found ${#bucket_uris[@]} expired buckets. Will attempt to delete at the most $(( MAX_BUCKET_DELETION_BATCHES * BUCKET_DELETION_BATCH_SIZE )) buckets."

    local batch_count=0
    local start_index=0
    local total_buckets=${#bucket_uris[@]}

    while [[ $start_index -lt $total_buckets ]]; do
        if [[ $batch_count -ge $MAX_BUCKET_DELETION_BATCHES ]]; then
            log_info "Reached maximum batch limit ($MAX_BUCKET_DELETION_BATCHES). Stopping cleanup."
            break
        fi

        # Calculate end index for the current batch
        local end_index=$((start_index + BUCKET_DELETION_BATCH_SIZE))
        if [[ $end_index -gt $total_buckets ]]; then
            end_index=$total_buckets
        fi

        # Extract batch slice
        # Note: Bash array slicing syntax is ${array[@]:start:length}
        local length=$((end_index - start_index))
        local batch=("${bucket_uris[@]:$start_index:$length}")

        batch_count=$((batch_count + 1))
        log_info "Deleting batch $batch_count (buckets $((start_index + 1)) to $end_index)..."

        # Delete batch
        if ! gcloud storage rm -r "${batch[@]}" --no-user-output-enabled --verbosity=error; then
            log_error "Failed to delete batch $batch_count. Aborting cleanup."
            return 1
        fi

        start_index=$end_index
    done

    log_info "Bucket cleanup complete."
}

# Get command of the PID and check if it contains the string. Kill if it does.
safe_kill() {
  local pid=$1
  local str=$2
  local cmd

  if [[ -n "$pid" && -n "$str" ]] && cmd=$(ps -p "$pid" -o cmd=) && [[ "$cmd" == *"$str"* ]]; then
    kill "$pid"
  else
    return 1
  fi
}

# Cleanup ensures each of the buckets created is destroyed and the temp files are cleaned up.
clean_up() {
  if ${TRACK_RESOURCE_USAGE}; then
    if ! safe_kill "$RESOURCE_USAGE_PID" "resource_usage.sh"; then
      log_error "Failed to stop resource usage collection process (or it's already stopped)"
    else
      log_info "Resource usage collection process stopped."
    fi
  fi
  if [ -n "${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}" ] && [ -d "${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}" ]; then
    log_info "Cleaning up GCSFuse build directory created by script: ${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}"
    rm -rf "${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}"
  fi
  cleanup_expired_buckets
  if ! rm -rf /tmp/"${TMP_PREFIX}"*; then 
    log_error "Failed to delete temporary files"
  else 
    log_info "Successfully cleaned up temporary files"
  fi
}

# Helper method to process any of the background process and
# returns exit status of waited pid.
process_any_pid() {
  local -n cmds_by_pid_ref="$1"
  local waited_pid
  local pid_status # To store the exit status of the waited pid

  wait -n -p waited_pid # waited_pid gets the PID, $? gets the status
  pid_status=$?

  unset "cmds_by_pid_ref[$waited_pid]"
  if [[ "$pid_status" -ne 0 ]]; then
    return 1
  fi
  return 0
}

# run_parallel: Executes commands in parallel based on a template and substitutes.
#   The function returns a non-zero exit status if any of the parallel commands fail.
#
# Usage: run_parallel "parallelism" "command_template_with_@" "substitute1" "substitute2" ...
#   First argument is extent of parallelism for this command.
#   Second argument is the command template with single @.
#   Rest of the arguments are values that would be substituted in the command template.
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
  local -A cmds_by_pid=()
  local overall_exit_code=0 parallel_cmd pid
  # Launch parallel commands in the background based on parallelism.
  for arg in "$@"; do
    parallel_cmd="${cmd_template//@/$arg}"
    eval "$parallel_cmd" &
    pid=$!
    cmds_by_pid["$pid"]="$parallel_cmd"
    if [[ ${#cmds_by_pid[@]} -eq $parallelism ]]; then
      process_any_pid "cmds_by_pid"
      overall_exit_code=$((overall_exit_code || $? ))
    fi
  done
  # Process any remaining PIDs
  while [[ ${#cmds_by_pid[@]} -gt 0 ]]; do
      process_any_pid "cmds_by_pid"
      overall_exit_code=$((overall_exit_code || $? ))
  done
  return $overall_exit_code
}

# Helper method that creates a bucket and then runs the test package.
create_bucket_and_run_test() {
  if [[ $# -ne 2 ]]; then
    log_error_locked "create_bucket_and_run_test() called with incorrect number of arguments."
    return 1
  fi
  local package_name="$1"
  local bucket_type="$2"

  if ! bucket_name=$(create_bucket "$package_name" "$bucket_type"); then
    log_error_locked "Failed to create bucket of type ${bucket_type} for package ${package_name}. Bucket creation output: ${bucket_name}"
    return 1
  fi
  test_package "$package_name" "$bucket_name" "$bucket_type"
}

# Helper method to executes e2e test package.
test_package() {
  if [[ $# -ne 3 ]]; then
    log_error_locked "test_package() called with incorrect number of arguments."
    return 1
  fi
  local package_name="$1"
  local bucket_name="$2"
  local bucket_type="$3"

  # Build go package test command.
  local go_test_cmd_parts=("GODEBUG=asyncpreemptoff=1" "go" "test" "-v" "-timeout=${INTEGRATION_TEST_PACKAGE_TIMEOUT_IN_MINS}m" "${INTEGRATION_TEST_PACKAGE_DIR}/${package_name}")
  if ${SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE}; then
    go_test_cmd_parts+=("-short")
  fi
  if [[ "$package_name" == "benchmarking" ]]; then
    go_test_cmd_parts+=("-bench=." "-benchtime=100x")
  fi
  # Test Binary flags after this.
  go_test_cmd_parts+=("-args" "--integrationTest" "--testbucket=${bucket_name}")
  if ${TEST_INSTALLED_PACKAGE}; then
    go_test_cmd_parts+=("--testInstalledPackage")
  fi
  if ${RUN_TESTS_WITH_PRESUBMIT_FLAG}; then
    go_test_cmd_parts+=("--presubmit")
  fi
  if [[ "$bucket_type" == "$ZONAL" ]]; then
    go_test_cmd_parts+=("--zonal")
  fi
  if ${RUN_TEST_ON_TPC_ENDPOINT}; then
    go_test_cmd_parts+=("--testOnTPCEndPoint")
  fi
  if [[ -n "$BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR" ]]; then 
    go_test_cmd_parts+=("--gcsfuse_prebuilt_dir=${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}")
  fi

  local go_test_cmd test_package_log_file start=$SECONDS exit_code=0 
  # Use printf %q to quote each argument safely for eval
  # This ensures spaces and special characters within arguments are handled correctly.
  go_test_cmd=$(printf "%q " "${go_test_cmd_parts[@]}")  
  test_package_log_file=$(mktemp "/tmp/${TMP_PREFIX}_${package_name}_${bucket_type}_log.XXXXXX")
  # Run the package test command and capture log output with runtime stats.
  log_info "Started running test package [$package_name] for bucket type [$bucket_type] with bucket name [$bucket_name]"
  if ! eval "$go_test_cmd" > "$test_package_log_file" 2>&1; then
    exit_code=1
    log_info "Failed test package [$package_name] for bucket type [$bucket_type]"
  else
    log_info "Passed test package [$package_name] for bucket type [$bucket_type]"
  fi
  local end=$SECONDS

  # Add the package stats to the file.
  echo "${package_name} ${bucket_type} ${exit_code} ${start} ${end}" >> "$PACKAGE_RUNTIME_STATS"
  # Generate Kokoro artifacts(log) files.
  generate_test_log_artifacts "$test_package_log_file" "$package_name" "$bucket_type"
  return "$exit_code"
}

# Helper method to generate Kokoro artifacts(log) files when building in Kokoro environment.
generate_test_log_artifacts() {
  # If KOKORO_ARTIFACTS_DIR is not set, skip artifact generation.
  if ! $KOKORO_DIR_AVAILABLE; then
    return 0
  fi

  if [[ $# -ne 3 ]]; then
    log_error_locked "generate_test_log_artifacts() called with incorrect number of arguments."
    return 1
  fi

  local log_file="$1"
  local package_name="$2"
  local bucket_type="$3"

  if [ ! -f "$log_file" ]; then
    return 0
  fi

  local output_dir="${KOKORO_ARTIFACTS_DIR}/${bucket_type}/${package_name}"
  mkdir -p "$output_dir"
  local sponge_log_file="${output_dir}/sponge_log.log"
  local sponge_xml_file="${output_dir}/sponge_log.xml"

  cp "$log_file" "$sponge_log_file"
  
  echo '<?xml version="1.0" encoding="UTF-8"?>' > "${sponge_xml_file}"
  echo '<testsuites>' >> "${sponge_xml_file}"

  # Remove first 2 lines and last line from log.
  local report_log=$(cat "$log_file")
  # For benchmarking package, filter out benchmark results to avoid incorrect XML results.
  if [[ "$package_name" == "benchmarking" ]]; then
    report_log=$(echo "$report_log" | grep -v '^Benchmark_[^[:space:]]*$')
  fi

  echo "$report_log" | go-junit-report | sed '1,2d;$d' >> "${sponge_xml_file}"
  echo '</testsuites>' >> "${sponge_xml_file}"
  
  return 0
}

build_gcsfuse_once() {
  local build_output_dir # For the final gcsfuse binaries
  build_output_dir=$(mktemp -d -t gcsfuse_e2e_run_build_XXXXXX)
  log_info "GCSFuse binaries will be built in ${build_output_dir}/"

  local gcsfuse_src_dir
  # Determine GCSFuse source directory
  # If this script is in tools/integration_tests, project root is ../../
  SCRIPT_DIR_REALPATH=$(realpath "$(dirname "${BASH_SOURCE[0]}")")
  gcsfuse_src_dir=$(realpath "${SCRIPT_DIR_REALPATH}/../../")

  if [[ ! -f "${gcsfuse_src_dir}/go.mod" ]]; then
    log_error "Could not reliably determine GCSFuse project root from ${SCRIPT_DIR_REALPATH}. Expected go.mod at ${gcsfuse_src_dir}" >&2
    rm -rf "${build_output_dir}"
    exit 1
  fi
  log_info "Using GCSFuse source directory: ${gcsfuse_src_dir}"

  log_info "Building GCSFuse using 'go run ./tools/build_gcsfuse/main.go'..."
  (cd "${gcsfuse_src_dir}" && go run ./tools/build_gcsfuse/main.go . "${build_output_dir}" "0.0.0")
  if [ $? -ne 0 ]; then
    log_error "Building GCSFuse binaries using 'go run ./tools/build_gcsfuse/main.go' failed."
    rm -rf "${build_output_dir}" # Clean up created temp dir
    return 1
  fi

  # Set the directory path for use by the script (to form the go test flag)
  BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR="${build_output_dir}"
  log_info "GCSFuse binaries built by script in: ${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}"
  log_info "GCSFuse executable: ${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}/bin/gcsfuse"
  return 0
}

install_packages() {
  local os_id
  
  # Determine the absolute location of THIS script
  SCRIPT_DIR=$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")

  # Calculate the Repo Root
  REPO_ROOT="${SCRIPT_DIR}/../.."

  source "${REPO_ROOT}/perfmetrics/scripts/os_utils.sh"
  
  if ! os_id=$(get_os_id); then
    log_error "Failed to detect OS ID."
    exit 1
  fi
  log_info "Detected OS ID: $os_id"

  install_packages_by_os "$os_id" "python3" "gcc" "python3-dev" "python3-setuptools" "python3-crcmod" || {
    log_error "Failed to install required packages."
    exit 1
  }

  # Execute install_go.sh using the absolute path
  bash "${REPO_ROOT}/perfmetrics/scripts/install_go.sh" "$GO_VERSION"
  export PATH="/usr/local/go/bin:$PATH"
  
  # Install latest gcloud version.
  bash "${REPO_ROOT}/perfmetrics/scripts/install_latest_gcloud.sh"
  export PATH="/usr/local/google-cloud-sdk/bin:$PATH"
  export CLOUDSDK_PYTHON="$HOME/.local/python-3.11.9/bin/python3.11"
  export PATH="$HOME/.local/python-3.11.9/bin:$PATH"
  if ${KOKORO_DIR_AVAILABLE} ; then
    # Install go-junit-report to generate XML test reports from go logs.
    go install github.com/jstemmer/go-junit-report/v2@latest
    export PATH="$(go env GOPATH)/bin:$PATH"
  fi
}

# Generic function to run a group of E2E tests for a given bucket type.
# Args:
#   $1: Descriptive group name (e.g., "REGIONAL", "ZONAL", "TPC")
#   $2: Bucket type ("flat", "hns", "zonal")
#   $@: A list of test package names to run.
run_test_group() {
  local group_name="$1"
  local bucket_type="$2"
  shift 2
  local -a test_packages=("$@")
  local group_exit_code=0
  log_info_locked "Started running e2e tests for ${group_name} group (bucket type: ${bucket_type})."

  run_parallel "$PACKAGE_LEVEL_PARALLELISM" "create_bucket_and_run_test @ ${bucket_type}" "${test_packages[@]}"
  group_exit_code=$?

  if [ "$group_exit_code" -ne 0 ]; then
    log_error_locked "The e2e tests for ${group_name} group (bucket type: ${bucket_type}) FAILED."
    return 1
  fi
  log_info_locked "The e2e tests for ${group_name} group (bucket type: ${bucket_type}) successful."
  return 0
}

run_e2e_tests_for_emulator() {
  log_info_locked "Started running e2e tests for emulator."
  local emulator_test_log=$(mktemp "/tmp/${TMP_PREFIX}_emulator_test_log.XXXXXX")
  if ! ./tools/integration_tests/emulator_tests/emulator_tests.sh "$TEST_INSTALLED_PACKAGE" > "$emulator_test_log" 2>&1; then
    acquire_lock "$LOG_LOCK_FILE"
    log_error ""
    log_error "--- Emulator Tests Failed ---"
    cat "$emulator_test_log"
    release_lock "$LOG_LOCK_FILE"
    return 1
  fi
  log_info_locked "Emulator tests successful."
  return 0
}

main() {
  # Clean up everything on exit.
  trap clean_up EXIT
  log_info ""
  log_info "------ Upgrading gcloud and installing packages ------"
  log_info ""
  set -e
  install_packages
  set +e
  log_info "------ Upgrading gcloud and installing packages took $SECONDS seconds ------"

  log_info ""
  log_info "------ Started running E2E test packages ------"
  log_info ""

  # Decide whether to build GCSFuse based on RUN_E2E_TESTS_ON_PACKAGE
  if (! ${TEST_INSTALLED_PACKAGE} ) && ${BUILD_BINARY_IN_SCRIPT}; then
    log_info "TEST_INSTALLED_PACKAGE is not 'true' (value: '${TEST_INSTALLED_PACKAGE}') and BUILD_BINARY_IN_SCRIPT is 'true'."
    log_info "Building GCSFuse inside script..."
    if ! build_gcsfuse_once; then
        log_error "build_gcsfuse_once failed. Exiting."
        # The trap will handle cleanup
        exit 1
    fi
    log_info "Script built GCSFuse at: ${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}"
  fi

  # Reset SECONDS to 0
  SECONDS=0

  if ${TRACK_RESOURCE_USAGE}; then
    # Start collecting system resource usage in background.
    log_info "Starting resource usage collection process."
    ./tools/integration_tests/resource_usage.sh "COLLECT" "$RESOURCE_USAGE_FILE" &
    RESOURCE_USAGE_PID=$!
    log_info "Resource usage collection process started at PID: $RESOURCE_USAGE_PID"
  fi

  local pids=()
  local overall_exit_code=0
  if ${RUN_TESTS_WITH_ZONAL_BUCKET}; then
    run_test_group "ZONAL" "$ZONAL" "${TEST_PACKAGES_FOR_ZB[@]}" & pids+=($!)
  elif ${RUN_TEST_ON_TPC_ENDPOINT}; then
    # Override PROJECT_ID and BUCKET_LOCATION for TPC tests
    PROJECT_ID="$TPCZERO_PROJECT_ID"
    BUCKET_LOCATION="$TPC_BUCKET_LOCATION"
    run_test_group "TPC" "$HNS" "${TEST_PACKAGES_FOR_TPC[@]}" & pids+=($!)
    run_test_group "TPC" "$FLAT" "${TEST_PACKAGES_FOR_TPC[@]}" & pids+=($!)
  else
    run_test_group "REGIONAL" "$HNS" "${TEST_PACKAGES_FOR_RB[@]}" & pids+=($!)
    run_test_group "REGIONAL" "$FLAT" "${TEST_PACKAGES_FOR_RB[@]}" & pids+=($!)
    run_e2e_tests_for_emulator & pids+=($!) # Emulator tests are a separate group
  fi
  # Wait for all background processes to complete and aggregate their exit codes
  for pid in "${pids[@]}"; do
    wait "$pid"
    overall_exit_code=$((overall_exit_code || $?))
  done
  elapsed_min=$(((SECONDS + 60) / 60))
  log_info "------ E2E test packages complete run took ${elapsed_min} minutes ------"
  log_info ""

  # Print package runtime stats table.
  ./tools/integration_tests/create_package_runtime_table.sh "$PACKAGE_RUNTIME_STATS"

  if ${TRACK_RESOURCE_USAGE}; then
    # Kill resource usage background PID and print resource usage.
    log_info "Stopping resource usage collection process: $RESOURCE_USAGE_PID"
    if safe_kill "$RESOURCE_USAGE_PID" "resource_usage.sh"; then
      log_info "Resource usage collection process stopped."
      ./tools/integration_tests/resource_usage.sh "PRINT" "$RESOURCE_USAGE_FILE"
    else
      log_error "Failed to stop resource usage collection process (or it's already stopped)"
    fi
  fi
  exit $overall_exit_code
}

#Main method to run script
main
