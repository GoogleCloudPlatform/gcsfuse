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

# --- Script Configuration & Constants ---
# Exit immediately if a command exits with a non-zero status.
set -e
# Treat unset variables as an error when substituting.
set -u
# Pipestatus: a pipeline's return status is the value of the last command to
# exit with a non-zero status, or zero if all commands exit successfully.
set -o pipefail

# --- Logging Helpers ---
# Prepends [INFO], [WARN], or [ERROR] and a timestamp to messages.
log_info() { echo "[INFO] $(date '+%Y-%m-%d %H:%M:%S') - $1"; }
log_warning() { echo "[WARN] $(date '+%Y-%m-%d %H:%M:%S') - $1"; }
log_error() { >&2 echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') - $1"; }

# --- Default Values & Constants ---
DEFAULT_INTEGRATION_TEST_TIMEOUT_MINS=90
RANDOM_STRING_LENGTH=5
GCP_PROJECT_ID_TEST_ML="gcs-fuse-test-ml"
GCP_PROJECT_ID_HNS="gcs-fuse-test"
GOLANG_GRPC_TEST_BUCKET_PREFIX="golang-grpc-test-" # Whitelisted for grpc tests

# Base directory for integration tests
INTEGRATION_TESTS_BASE_DIR="./tools/integration_tests"
LOG_OUTPUT_DIR="/tmp"
JUNIT_REPORT_DIR="/tmp" # For Sponge/ResultsStore

# Test directory arrays
# These tests can be run in parallel.
TEST_DIRS_PARALLEL=(
  "monitoring" "local_file" "log_rotation" "mounting" "read_cache"
  "gzip" "write_large_files" "list_large_dir" "rename_dir_limit"
  "read_large_files" "explicit_dir" "implicit_dir" "interrupt"
  "operations" "kernel_list_cache" "concurrent_operations"
  "benchmarking" "mount_timeout" "stale_handle" "negative_stat_cache"
  "streaming_writes"
)

# These tests must be run sequentially (e.g., due to bucket permission changes).
TEST_DIRS_NON_PARALLEL=(
  "readonly" "managed_folders" "readonly_creds"
)

# Subset of TEST_DIRS_PARALLEL for Zonal Bucket (ZB) runs.
TEST_DIRS_PARALLEL_ZB=(
  "benchmarking" "explicit_dir" "gzip" "implicit_dir" "interrupt"
  "kernel_list_cache" "local_file" "log_rotation" "monitoring"
  "mount_timeout" "mounting" "negative_stat_cache" "operations"
  "read_cache" "read_large_files" "rename_dir_limit" "stale_handle"
  "streaming_writes" "write_large_files" "unfinalized_object"
)

# Subset of TEST_DIRS_NON_PARALLEL for Zonal Bucket (ZB) runs.
TEST_DIRS_NON_PARALLEL_ZB=(
  "concurrent_operations" "list_large_dir" "managed_folders"
  "readonly" "readonly_creds"
)

# --- Global Variables (will be set by argument parsing) ---
RUN_E2E_ON_PACKAGE=""
SKIP_NON_ESSENTIAL_TESTS=""
BUCKET_LOCATION_CONFIG=""
RUN_ON_TPC_ENDPOINT=""
PRESUBMIT_RUN_FLAG_ARG=""
RUN_ON_ZONAL_BUCKET_FLAG=""

# Calculated variables
GO_TEST_EXTRA_FLAGS="" # For -short, -presubmit
INTEGRATION_TEST_TIMEOUT_DURATION=""
ALL_TEST_LOGS_TEMP_FILE="" # File to store paths of all individual test logs

# --- Function Definitions ---

# Parse and validate script arguments
parse_and_validate_arguments() {
  if [[ "$#" -lt 3 ]]; then
    log_error "Incorrect number of arguments. Expected at least 3."
    echo "Usage: $0 <RUN_E2E_TESTS_ON_PACKAGE_BOOL> <SKIP_NON_ESSENTIAL_TESTS_BOOL> <BUCKET_LOCATION_STRING> [RUN_TEST_ON_TPC_ENDPOINT_BOOL] [PRESUBMIT_RUN_FLAG_BOOL] [RUN_TESTS_WITH_ZONAL_BUCKET_BOOL]"
    exit 1
  fi

  RUN_E2E_ON_PACKAGE="${1}" # true or false
  SKIP_NON_ESSENTIAL_TESTS="${2}" # true or false
  BUCKET_LOCATION_CONFIG="${3}" # e.g., us-west1
  RUN_ON_TPC_ENDPOINT="${4:-false}" # Default to false
  PRESUBMIT_RUN_FLAG_ARG="${5:-false}" # Default to false
  RUN_ON_ZONAL_BUCKET_FLAG="${6:-false}" # Default to false

  local current_timeout_mins=${DEFAULT_INTEGRATION_TEST_TIMEOUT_MINS}

  if [[ "${SKIP_NON_ESSENTIAL_TESTS}" == "true" ]]; then
    GO_TEST_EXTRA_FLAGS+="-short "
    log_info "Skipping non-essential tests (-short flag enabled)."
    current_timeout_mins=$((current_timeout_mins - 20))
  fi

  if [[ "${PRESUBMIT_RUN_FLAG_ARG}" == "true" ]]; then
    GO_TEST_EXTRA_FLAGS+="-presubmit "
    log_info "Presubmit run detected (-presubmit flag enabled)."
    current_timeout_mins=$((current_timeout_mins - 10))
  fi

  INTEGRATION_TEST_TIMEOUT_DURATION="${current_timeout_mins}m"
  log_info "Integration test timeout set to: ${INTEGRATION_TEST_TIMEOUT_DURATION}"

  if [[ "${RUN_ON_ZONAL_BUCKET_FLAG}" == "true" ]]; then
    if [[ "${BUCKET_LOCATION_CONFIG}" != "us-west4" && "${BUCKET_LOCATION_CONFIG}" != "us-central1" ]]; then
      log_error "For zonal bucket runs, BUCKET_LOCATION must be 'us-west4' or 'us-central1'. Passed: ${BUCKET_LOCATION_CONFIG}"
      exit 1
    fi
    log_info "Configured to run tests specifically on Zonal Buckets in ${BUCKET_LOCATION_CONFIG}."
  fi

  if [[ "${RUN_E2E_ON_PACKAGE}" != "true" && "${RUN_E2E_ON_PACKAGE}" != "false" ]]; then
    log_error "Invalid value for RUN_E2E_TESTS_ON_PACKAGE: ${RUN_E2E_ON_PACKAGE}. Expected 'true' or 'false'."
    exit 1
  fi
   if [[ "${SKIP_NON_ESSENTIAL_TESTS}" != "true" && "${SKIP_NON_ESSENTIAL_TESTS}" != "false" ]]; then
    log_error "Invalid value for SKIP_NON_ESSENTIAL_TESTS: ${SKIP_NON_ESSENTIAL_TESTS}. Expected 'true' or 'false'."
    exit 1
  fi
  # Further validation for other boolean flags can be added here.
}

# Upgrade gcloud and install necessary components
upgrade_gcloud_components() {
  log_info "Updating apt-get..."
  sudo apt-get update -qq
  log_info "Checking current gcloud version..."
  gcloud version
  log_info "Downloading latest Google Cloud SDK..."
  wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
  log_info "Installing Google Cloud SDK..."
  sudo tar xzf gcloud.tar.gz && sudo cp -r google-cloud-sdk /usr/local && sudo rm -rf google-cloud-sdk gcloud.tar.gz
  sudo /usr/local/google-cloud-sdk/install.sh --quiet
  export PATH="/usr/local/google-cloud-sdk/bin:${PATH}" # Ensure it's in PATH for current session
  echo 'export PATH=/usr/local/google-cloud-sdk/bin:$PATH' >> "${HOME}/.bashrc"
  log_info "New gcloud version:"
  gcloud version
  log_info "Updating gcloud components and installing alpha..."
  sudo /usr/local/google-cloud-sdk/bin/gcloud components update --quiet
  sudo /usr/local/google-cloud-sdk/bin/gcloud components install alpha --quiet
  log_info "gcloud setup complete."
}

# Install Go and other dependencies
install_required_packages() {
  local architecture
  architecture=$(dpkg --print-architecture)
  log_info "Installing Go 1.24.0 for ${architecture}..."
  wget -O go_tar.tar.gz https://go.dev/dl/go1.24.0.linux-"${architecture}".tar.gz -q
  sudo rm -rf /usr/local/go && sudo tar -xzf go_tar.tar.gz -C /usr/local && sudo rm go_tar.tar.gz
  export PATH="${PATH}:/usr/local/go/bin" # For current session
  log_info "Installing Python3, setuptools, gcc, and crcmod..."
  sudo apt-get install -y -qq python3 gcc python3-dev python3-setuptools python3-crcmod
  # Consider installing gotestsum here if it's a hard dependency for JUnit reports
  go install gotest.tools/gotestsum@latest
  log_info "Package installation complete."
}

# Combined environment setup
prepare_execution_environment() {
  log_info "Starting environment preparation..."
  upgrade_gcloud_components
  install_required_packages
  log_info "Environment preparation finished."
}

# Generate a unique bucket name
generate_bucket_name() {
  local prefix="${1}"
  local random_suffix
  random_suffix=$(tr -dc 'a-z0-9' < /dev/urandom | head -c "${RANDOM_STRING_LENGTH}")
  echo "${prefix}$(date +%Y%m%d-%H%M%S)-${random_suffix}"
}

# Create a standard GCS bucket
create_standard_bucket() {
  local bucket_prefix="${1}"
  local bucket_name
  bucket_name=$(generate_bucket_name "${bucket_prefix}")
  log_info "Creating standard bucket: gs://${bucket_name} in ${BUCKET_LOCATION_CONFIG} for project ${GCP_PROJECT_ID_TEST_ML}..."
  gcloud alpha storage buckets create "gs://${bucket_name}" \
    --project="${GCP_PROJECT_ID_TEST_ML}" \
    --location="${BUCKET_LOCATION_CONFIG}" \
    --uniform-bucket-level-access \
    -q # Quiet mode for gcloud
  echo "${bucket_name}"
}

# Create an HNS-enabled GCS bucket
create_hns_bucket() {
  local bucket_name
  # HNS buckets use a different project and a specific prefix for whitelisting.
  bucket_name=$(generate_bucket_name "${GOLANG_GRPC_TEST_BUCKET_PREFIX}gcsfuse-e2e-hns-")
  log_info "Creating HNS bucket: gs://${bucket_name} in ${BUCKET_LOCATION_CONFIG} for project ${GCP_PROJECT_ID_HNS}..."
  gcloud alpha storage buckets create "gs://${bucket_name}" \
    --project="${GCP_PROJECT_ID_HNS}" \
    --location="${BUCKET_LOCATION_CONFIG}" \
    --uniform-bucket-level-access \
    --enable-hierarchical-namespace \
    -q
  echo "${bucket_name}"
}

# Create a Zonal GCS bucket
create_zonal_bucket() {
  local bucket_name
  local region="${BUCKET_LOCATION_CONFIG}" # For ZB, BUCKET_LOCATION_CONFIG is the region
  local zone="${region}-a" # Default to zone 'a'

  bucket_name=$(generate_bucket_name "gcsfuse-e2e-zb-")
  log_info "Creating Zonal bucket: gs://${bucket_name} in project ${GCP_PROJECT_ID_TEST_ML}, region ${region}, placement-zone ${zone}..."
  gcloud alpha storage buckets create "gs://${bucket_name}" \
    --project="${GCP_PROJECT_ID_TEST_ML}" \
    --location="${region}" \
    --placement="${zone}" \
    --default-storage-class="RAPID" \
    --uniform-bucket-level-access \
    --enable-hierarchical-namespace \
    -q
  echo "${bucket_name}"
}

# Execute a single Go test package
# Arguments:
#   $1: Test directory name (e.g., "operations")
#   $2: Bucket name for the test
#   $3: Zonal flag ("true" or "false")
#   $4: Benchmark flags (e.g., "-bench=. -benchtime=100x" or empty string)
#   $5: Base for log file name
#   $6: Base for JUnit report file name
execute_test_package() {
  local test_dir_name="${1}"
  local test_bucket_name="${2}"
  local zonal_flag="${3}"
  local benchmark_run_flags="${4}"
  local log_file_base="${5}"
  local junit_file_base="${6}"

  local test_path="${INTEGRATION_TESTS_BASE_DIR}/${test_dir_name}"
  local log_file="${LOG_OUTPUT_DIR}/${log_file_base}.log"
  local junit_report_file="${JUNIT_REPORT_DIR}/${junit_file_base}.xml"

  echo "${log_file}" >> "${ALL_TEST_LOGS_TEMP_FILE}" # Record log file path

  local test_command_args=(
    "${test_path}"
    ${GO_TEST_EXTRA_FLAGS} # Expands to -short, -presubmit if set
    "--zonal=${zonal_flag}"
    "--integrationTest"
    "-v"
    "--testbucket=${test_bucket_name}"
    "--testInstalledPackage=${RUN_E2E_ON_PACKAGE}"
    "-timeout" "${INTEGRATION_TEST_TIMEOUT_DURATION}"
    "-p" "1" # Run tests within a package sequentially
  )
  if [[ -n "${benchmark_run_flags}" ]]; then
    #shellcheck disable=SC2206 # Intentionally splitting benchmark_run_flags
    test_command_args+=(${benchmark_run_flags})
  fi

  log_info "Executing test: ${test_dir_name} (Zonal: ${zonal_flag}) on bucket ${test_bucket_name}. Log: ${log_file}"

  # Use gotestsum if test_dir_name is "operations" for JUnit output, otherwise standard go test
  # Ensure gotestsum is installed: go install gotest.tools/gotestsum@latest
  local final_exit_code=0
  if [[ "${test_dir_name}" == "operations" && -x "$(command -v gotestsum)" ]]; then
    log_info "Using gotestsum for JUnit report: ${junit_report_file}"
    GODEBUG=asyncpreemptoff=1 gotestsum --junitfile "${junit_report_file}" -- "${test_command_args[@]}" > "${log_file}" 2>&1
    final_exit_code=$?
  elif [[ "${test_dir_name}" == "operations" ]];
  then
    log_warning "gotestsum not found. Skipping JUnit report for ${test_dir_name}. Falling back to standard go test."
    GODEBUG=asyncpreemptoff=1 go test "${test_command_args[@]}" > "${log_file}" 2>&1
    final_exit_code=$?
  else
    GODEBUG=asyncpreemptoff=1 go test "${test_command_args[@]}" > "${log_file}" 2>&1
    final_exit_code=$?
  fi
  return ${final_exit_code}
}


# Run a suite of tests (either in parallel or sequentially)
# Arguments:
#   $1: Reference to the array of test directories (pass as TEST_DIRS_ARRAY_NAME)
#   $2: Bucket name for these tests
#   $3: "parallel" or "sequential"
#   $4: Zonal flag ("true" or "false")
run_test_suite() {
  local -n test_dirs_ref=$1 # Use nameref for array passing
  local suite_bucket_name=$2
  local execution_mode=$3 # "parallel" or "sequential"
  local zonal_run_flag=$4
  local overall_suite_exit_code=0
  declare -A pids # Associative array for PIDs if running in parallel

  log_info "Starting test suite on bucket ${suite_bucket_name}. Mode: ${execution_mode}. Zonal: ${zonal_run_flag}."

  for test_dir in "${test_dirs_ref[@]}"; do
    local current_benchmark_flags=""
    if [[ "${test_dir}" == "benchmarking" ]]; then
      current_benchmark_flags="-bench=. -benchtime=100x"
    fi

    local log_file_name_base="${test_dir}_${suite_bucket_name}"
    local junit_file_name_base="${test_dir}_${suite_bucket_name}" # For execute_test_package

    if [[ "${execution_mode}" == "parallel" ]]; then
      execute_test_package "${test_dir}" "${suite_bucket_name}" "${zonal_run_flag}" "${current_benchmark_flags}" "${log_file_name_base}" "${junit_file_name_base}" &
      pids[${test_dir}]=$!
      log_info "Queued parallel test: ${test_dir} (PID: ${pids[${test_dir}]})"
    else # sequential
      if ! execute_test_package "${test_dir}" "${suite_bucket_name}" "${zonal_run_flag}" "${current_benchmark_flags}" "${log_file_name_base}" "${junit_file_name_base}"; then
        log_error "Sequential test FAILED: ${test_dir} on bucket ${suite_bucket_name} (Zonal: ${zonal_run_flag})"
        overall_suite_exit_code=1 # Mark failure
      else
        log_info "Sequential test PASSED: ${test_dir} on bucket ${suite_bucket_name} (Zonal: ${zonal_run_flag})"
      fi
    fi
  done

  if [[ "${execution_mode}" == "parallel" ]]; then
    log_info "Waiting for all parallel tests in suite for bucket ${suite_bucket_name} to complete..."
    for test_dir_pid_key in "${!pids[@]}"; do
      local pid_to_wait_on="${pids[${test_dir_pid_key}]}"
      if wait "${pid_to_wait_on}"; then
        log_info "Parallel test PASSED: ${test_dir_pid_key} (PID: ${pid_to_wait_on})"
      else
        log_error "Parallel test FAILED: ${test_dir_pid_key} (PID: ${pid_to_wait_on})"
        overall_suite_exit_code=1 # Mark failure
      fi
    done
  fi

  log_info "Test suite finished for bucket ${suite_bucket_name}. Overall status: ${overall_suite_exit_code}"
  return ${overall_suite_exit_code}
}


# Orchestrates E2E tests for flat buckets
run_e2e_tests_for_flat_bucket_type() {
  local overall_exit_code=0
  log_info "Starting E2E tests for FLAT bucket type..."

  local non_parallel_bucket_name
  non_parallel_bucket_name=$(create_standard_bucket "${GOLANG_GRPC_TEST_BUCKET_PREFIX}gcsfuse-np-e2e-")
  log_info "Flat bucket for non-parallel tests: ${non_parallel_bucket_name}"
  echo "${non_parallel_bucket_name}" >> "${BUCKET_NAMES_TO_DELETE_FILE}"

  local parallel_bucket_name
  parallel_bucket_name=$(create_standard_bucket "${GOLANG_GRPC_TEST_BUCKET_PREFIX}gcsfuse-p-e2e-")
  log_info "Flat bucket for parallel tests: ${parallel_bucket_name}"
  echo "${parallel_bucket_name}" >> "${BUCKET_NAMES_TO_DELETE_FILE}"

  # Run tests suites
  run_test_suite TEST_DIRS_PARALLEL "${parallel_bucket_name}" "parallel" "false" &
  local parallel_tests_pid=$!

  run_test_suite TEST_DIRS_NON_PARALLEL "${non_parallel_bucket_name}" "sequential" "false" &
  local non_parallel_tests_pid=$!

  wait "${parallel_tests_pid}"
  local parallel_exit_code=$?
  wait "${non_parallel_tests_pid}"
  local non_parallel_exit_code=$?

  if [[ ${parallel_exit_code} -ne 0 || ${non_parallel_exit_code} -ne 0 ]]; then
    log_error "One or more test suites FAILED for FLAT bucket type."
    overall_exit_code=1
  else
    log_info "All test suites PASSED for FLAT bucket type."
  fi
  return ${overall_exit_code}
}

# Orchestrates E2E tests for HNS buckets
run_e2e_tests_for_hns_bucket_type(){
  local overall_exit_code=0
  log_info "Starting E2E tests for HNS bucket type..."

  local hns_parallel_bucket
  hns_parallel_bucket=$(create_hns_bucket)
  log_info "HNS bucket for parallel tests: ${hns_parallel_bucket}"
  echo "${hns_parallel_bucket}" >> "${BUCKET_NAMES_TO_DELETE_FILE}"

  local hns_non_parallel_bucket
  hns_non_parallel_bucket=$(create_hns_bucket)
  log_info "HNS bucket for non-parallel tests: ${hns_non_parallel_bucket}"
  echo "${hns_non_parallel_bucket}" >> "${BUCKET_NAMES_TO_DELETE_FILE}"

  run_test_suite TEST_DIRS_PARALLEL "${hns_parallel_bucket}" "parallel" "false" &
  local parallel_hns_pid=$!
  run_test_suite TEST_DIRS_NON_PARALLEL "${hns_non_parallel_bucket}" "sequential" "false" &
  local non_parallel_hns_pid=$!

  wait ${parallel_hns_pid}
  local parallel_hns_exit_code=$?
  wait ${non_parallel_hns_pid}
  local non_parallel_hns_exit_code=$?

  if [[ ${parallel_hns_exit_code} -ne 0 || ${non_parallel_hns_exit_code} -ne 0 ]]; then
    log_error "One or more test suites FAILED for HNS bucket type."
    overall_exit_code=1
  else
    log_info "All test suites PASSED for HNS bucket type."
  fi
  return ${overall_exit_code}
}

# Orchestrates E2E tests for Zonal buckets
run_e2e_tests_for_zonal_bucket_type(){
  local overall_exit_code=0
  log_info "Starting E2E tests for ZONAL bucket type..."

  local zonal_parallel_bucket
  zonal_parallel_bucket=$(create_zonal_bucket)
  log_info "Zonal bucket for parallel tests: ${zonal_parallel_bucket}"
  echo "${zonal_parallel_bucket}" >> "${BUCKET_NAMES_TO_DELETE_FILE}"

  local zonal_non_parallel_bucket
  zonal_non_parallel_bucket=$(create_zonal_bucket)
  log_info "Zonal bucket for non-parallel tests: ${zonal_non_parallel_bucket}"
  echo "${zonal_non_parallel_bucket}" >> "${BUCKET_NAMES_TO_DELETE_FILE}"

  # Note: Using ZB-specific test arrays and zonal_run_flag="true"
  run_test_suite TEST_DIRS_PARALLEL_ZB "${zonal_parallel_bucket}" "parallel" "true" &
  local parallel_zonal_pid=$!
  run_test_suite TEST_DIRS_NON_PARALLEL_ZB "${zonal_non_parallel_bucket}" "sequential" "true" &
  local non_parallel_zonal_pid=$!

  wait ${parallel_zonal_pid}
  local parallel_zonal_exit_code=$?
  wait ${non_parallel_zonal_pid}
  local non_parallel_zonal_exit_code=$?

  if [[ ${parallel_zonal_exit_code} -ne 0 || ${non_parallel_zonal_exit_code} -ne 0 ]]; then
    log_error "One or more test suites FAILED for ZONAL bucket type."
    overall_exit_code=1
  else
    log_info "All test suites PASSED for ZONAL bucket type."
  fi
  return ${overall_exit_code}
}

# Run E2E tests on TPC endpoint (specifically 'operations' tests)
run_e2e_tests_for_tpc_endpoint() {
  local tpc_bucket_name="${1}"
  if [[ -z "${tpc_bucket_name}" ]]; then
    log_error "TPC bucket name is required for TPC tests."
    return 1
  fi
  echo "${tpc_bucket_name}" >> "${BUCKET_NAMES_TO_DELETE_FILE}" # Ensure it's cleaned up

  log_info "Preparing to run TPC tests on bucket: ${tpc_bucket_name}..."
  log_info "Cleaning bucket gs://${tpc_bucket_name} before TPC tests..."
  gcloud --verbosity=error storage rm -r "gs://${tpc_bucket_name}/*" || log_warning "No objects to delete in gs://${tpc_bucket_name} or bucket is empty."

  local tpc_test_exit_code=0
  local tpc_log_file_base="operations_tpc_${tpc_bucket_name}"
  local tpc_junit_file_base="operations_tpc_${tpc_bucket_name}" # For execute_test_package

  # TPC tests specifically target the 'operations' package.
  # The execute_test_package function expects a test directory name.
  # Here, we provide all necessary flags directly to go test or gotestsum.

  local tpc_test_command_args=(
    "${INTEGRATION_TESTS_BASE_DIR}/operations/..." # Explicitly target operations package
    "--testOnTPCEndPoint=${RUN_ON_TPC_ENDPOINT}"
    ${GO_TEST_EXTRA_FLAGS}
    "--zonal=false"
    "-p" "1"
    "--integrationTest"
    "-v"
    "--testbucket=${tpc_bucket_name}"
    "--testInstalledPackage=${RUN_E2E_ON_PACKAGE}"
    "-timeout" "${INTEGRATION_TEST_TIMEOUT_DURATION}"
  )

  local tpc_log_file="${LOG_OUTPUT_DIR}/${tpc_log_file_base}.log"
  local tpc_junit_report_file="${JUNIT_REPORT_DIR}/${tpc_junit_file_base}.xml"
  echo "${tpc_log_file}" >> "${ALL_TEST_LOGS_TEMP_FILE}"

  log_info "Running 'operations' E2E tests for TPC on bucket ${tpc_bucket_name}. Log: ${tpc_log_file}"
  if [[ -x "$(command -v gotestsum)" ]]; then
    log_info "Using gotestsum for JUnit report: ${tpc_junit_report_file}"
    GODEBUG=asyncpreemptoff=1 gotestsum --junitfile "${tpc_junit_report_file}" -- "${tpc_test_command_args[@]}" > "${tpc_log_file}" 2>&1
    tpc_test_exit_code=$?
  else
    log_warning "gotestsum not found. Skipping JUnit report for TPC operations test. Falling back to standard go test."
    GODEBUG=asyncpreemptoff=1 go test "${tpc_test_command_args[@]}" > "${tpc_log_file}" 2>&1
    tpc_test_exit_code=$?
  fi

  log_info "Cleaning bucket gs://${tpc_bucket_name} after TPC tests..."
  gcloud --verbosity=error storage rm -r "gs://${tpc_bucket_name}/*" || log_warning "No objects to delete in gs://${tpc_bucket_name} or bucket is empty post-test."

  if [[ ${tpc_test_exit_code} -ne 0 ]]; then
    log_error "TPC 'operations' tests FAILED on bucket ${tpc_bucket_name}."
    return 1
  fi
  log_info "TPC 'operations' tests PASSED on bucket ${tpc_bucket_name}."
  return 0
}


# Run E2E tests using the storage emulator
run_e2e_tests_with_emulator() {
  log_info "Starting E2E tests with emulator..."
  local emulator_script_path="${INTEGRATION_TESTS_BASE_DIR}/emulator_tests/emulator_tests.sh"
  if [[ ! -f "${emulator_script_path}" ]]; then
    log_error "Emulator test script not found at ${emulator_script_path}"
    return 1
  fi

  # The emulator script might have its own logging.
  # We'll capture its output to a main log file for emulator tests.
  local emulator_log_file="${LOG_OUTPUT_DIR}/emulator_tests_main.log"
  echo "${emulator_log_file}" >> "${ALL_TEST_LOGS_TEMP_FILE}"

  if "${emulator_script_path}" "${RUN_E2E_ON_PACKAGE}" > "${emulator_log_file}" 2>&1; then
    log_info "Emulator E2E tests PASSED."
    return 0
  else
    log_error "Emulator E2E tests FAILED. Check log: ${emulator_log_file}"
    return 1
  fi
}

# Cleanup: Delete GCS buckets listed in the tracking file
cleanup_created_buckets() {
  if [[ -f "${BUCKET_NAMES_TO_DELETE_FILE}" ]]; then
    log_info "Cleaning up created GCS buckets listed in ${BUCKET_NAMES_TO_DELETE_FILE}..."
    # Read each line, trim whitespace, and delete if not empty
    # Using a subshell to avoid issues with read in a loop modifying parent shell variables
    (
      while IFS= read -r bucket_name_to_delete || [[ -n "${bucket_name_to_delete}" ]]; do
        local trimmed_bucket_name
        trimmed_bucket_name=$(echo "${bucket_name_to_delete}" | xargs) # Trim whitespace
        if [[ -n "${trimmed_bucket_name}" ]]; then
          log_info "Attempting to delete bucket: gs://${trimmed_bucket_name}"
          if gcloud -q storage rm -r "gs://${trimmed_bucket_name}" --verbosity=none; then
            log_info "Successfully deleted bucket: gs://${trimmed_bucket_name}"
          else
            log_error "Failed to delete bucket: gs://${trimmed_bucket_name}. Manual cleanup might be required."
          fi
        fi
      done < "${BUCKET_NAMES_TO_DELETE_FILE}"
    )
    rm -f "${BUCKET_NAMES_TO_DELETE_FILE}"
    log_info "Bucket cleanup finished."
  else
    log_info "No bucket tracking file found (${BUCKET_NAMES_TO_DELETE_FILE}). Skipping bucket cleanup."
  fi
}

# Print content of all collected test logs
collate_and_print_test_logs() {
  if [[ -f "${ALL_TEST_LOGS_TEMP_FILE}" ]]; then
    log_info "--- Collating all test logs ---"
    # Using a subshell for the read loop
    (
      while IFS= read -r log_file_path || [[ -n "${log_file_path}" ]]; do
        local trimmed_log_path
        trimmed_log_path=$(echo "${log_file_path}" | xargs) # Trim
        if [[ -f "${trimmed_log_path}" ]]; then
          echo ""
          log_info "=== START: Log for ${trimmed_log_path} ==="
          cat "${trimmed_log_path}"
          log_info "=== END: Log for ${trimmed_log_path} ==="
          echo ""
        else
          log_warning "Log file not found: ${trimmed_log_path}"
        fi
      done < "${ALL_TEST_LOGS_TEMP_FILE}"
    )
    rm -f "${ALL_TEST_LOGS_TEMP_FILE}"
    log_info "--- Finished collating test logs ---"
  else
    log_info "No main log tracking file found (${ALL_TEST_LOGS_TEMP_FILE}). Cannot print individual logs."
  fi
}

# --- Main Script Execution ---
main() {
  # Initialize a file to track all buckets created by this script run
  # This ensures that the trap function knows which file to read from
  # mktemp ensures a unique filename to avoid collisions if script runs in parallel (though not recommended for this script)
  BUCKET_NAMES_TO_DELETE_FILE=$(mktemp "${LOG_OUTPUT_DIR}/gcsfuse_e2e_buckets_to_delete.XXXXXX")
  export BUCKET_NAMES_TO_DELETE_FILE # Make it available to the trap

  # Initialize a file to track paths of all individual test logs
  ALL_TEST_LOGS_TEMP_FILE=$(mktemp "${LOG_OUTPUT_DIR}/gcsfuse_e2e_all_test_logs.XXXXXX")
  export ALL_TEST_LOGS_TEMP_FILE

  # Trap to ensure cleanup runs on exit (normal or error)
  # shellcheck disable=SC2064 # BUCKET_NAMES_TO_DELETE_FILE is expanded when trap is set
  trap "log_info 'Executing cleanup trap...'; cleanup_created_buckets; collate_and_print_test_logs" EXIT SIGINT SIGTERM

  log_info "Starting E2E Test Script..."
  parse_and_validate_arguments "$@"

  # Environment setup (gcloud, packages)
  # This is a long operation, run it early.
  prepare_execution_environment

  local final_script_exit_code=0

  # Integration tests execution based on flags
  if [[ "${RUN_ON_ZONAL_BUCKET_FLAG}" == "true" ]]; then
    log_info "Executing E2E tests for ZONAL bucket configuration..."
    if ! run_e2e_tests_for_zonal_bucket_type; then
      log_error "E2E tests for ZONAL configuration FAILED."
      final_script_exit_code=1
    else
      log_info "E2E tests for ZONAL configuration PASSED."
    fi
  elif [[ "${RUN_ON_TPC_ENDPOINT}" == "true" ]]; then
    log_info "Executing E2E tests for TPC endpoint..."
    local tpc_overall_status=0
    # Run TPC tests for a flat bucket
    if ! run_e2e_tests_for_tpc_endpoint "gcsfuse-e2e-tests-tpc"; then
        log_error "TPC E2E tests for FLAT bucket FAILED."
        tpc_overall_status=1
    fi
    # Run TPC tests for an HNS bucket
    if ! run_e2e_tests_for_tpc_endpoint "gcsfuse-e2e-tests-tpc-hns"; then
        log_error "TPC E2E tests for HNS bucket FAILED."
        tpc_overall_status=1
    fi

    if [[ ${tpc_overall_status} -ne 0 ]]; then
        log_error "One or more TPC E2E test runs FAILED."
        final_script_exit_code=1
    else
        log_info "All TPC E2E tests PASSED."
    fi
    # TPC runs are exclusive, so exit after them.
    # The trap will handle cleanup and log collation.
    exit ${final_script_exit_code}
  else
    # Standard run: Flat, HNS, and Emulator tests
    log_info "Executing E2E tests for standard (Flat, HNS, Emulator) configurations..."
    declare -A test_flow_pids
    declare -A test_flow_status
    local any_standard_flow_failed=0

    run_e2e_tests_for_hns_bucket_type &
    test_flow_pids[HNS]=$!
    run_e2e_tests_for_flat_bucket_type &
    test_flow_pids[FLAT]=$!
    run_e2e_tests_with_emulator &
    test_flow_pids[EMULATOR]=$!

    log_info "Waiting for HNS, Flat, and Emulator test flows to complete..."
    for flow_type in "${!test_flow_pids[@]}"; do
      wait "${test_flow_pids[${flow_type}]}"
      test_flow_status[${flow_type}]=$?
      if [[ ${test_flow_status[${flow_type}]} -ne 0 ]]; then
        log_error "Test flow for ${flow_type} FAILED with status ${test_flow_status[${flow_type}]}."
        any_standard_flow_failed=1
      else
        log_info "Test flow for ${flow_type} PASSED."
      fi
    done

    if [[ ${any_standard_flow_failed} -ne 0 ]]; then
        log_error "One or more standard E2E test flows (Flat, HNS, Emulator) FAILED."
        final_script_exit_code=1
    else
        log_info "All standard E2E test flows (Flat, HNS, Emulator) PASSED."
    fi
  fi

  log_info "E2E Test Script finished with overall status: ${final_script_exit_code}."
  # Cleanup (bucket deletion and log printing) will be handled by the EXIT trap.
  exit ${final_script_exit_code}
}

# Script Entry Point
# Pass all script arguments to the main function
main "$@"