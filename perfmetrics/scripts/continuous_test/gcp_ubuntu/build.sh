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

# Exit on error, treat unset variables as errors, and propagate pipeline errors.
set -euo pipefail

# -----------------------------------------------------------------
# Environment Setup
# -----------------------------------------------------------------
sudo apt-get update
echo "Installing git"
sudo apt-get install -y git

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"

# Get the branch name that was cloned by Kokoro
branchName=$(git branch --format='%(refname:short)' | grep -v 'HEAD' | head -n 1)

# Utilize Louhi custom commit checkouts if provided
if [[ -n "${_COMMIT_HASH:-}" ]]; then
  echo "Louhi environment variable _COMMIT_HASH detected: ${_COMMIT_HASH}"
  echo "Checking out custom commit..."
  git checkout "${_COMMIT_HASH}"
  commitId="${_COMMIT_HASH}"
else
  echo "No _COMMIT_HASH detected. Proceeding with default branch checkout logic."
  # Get the commitId. Build gcsfuse and run.
  # - Automated daily runs (initiated by Kokoro scheduler) will run on the last commit of yesterday on the master branch.
  # - Manual runs (initiated by users) will run on the latest commit of the branch (master or feature branch) provided in the manual trigger.
  if [[ "${KOKORO_BUILD_INITIATOR:-}" == "kokoro" ]]; then
    commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
  else
    commitId=$(git log -n 1 --pretty=%H)
  fi
fi
echo "Running tests on branch: ${branchName} at commit ID: ${commitId}"

# -----------------------------------------------------------------
# Helper function to calculate and print execution time
# -----------------------------------------------------------------
print_duration() {
  local task_name="$1"
  local start_time="$2"
  local end_time=$SECONDS
  local duration=$((end_time - start_time))
  echo "================================================================="
  echo "⏱️  EXECUTION TIME - ${task_name}: ${duration} seconds"
  echo "================================================================="
}

# Record start time of the entire script
TOTAL_START=$SECONDS

# Trap to always print total execution time on exit
exit_handler() {
  print_duration "Total Execution Time" "$TOTAL_START"
}
trap exit_handler EXIT

# =================================================================
# 1) DISTRIBUTED READ BENCHMARK
# =================================================================
if [ "${BENCHMARK_TYPE:-}" == "distributed_benchmark_read" ]; then
  TOOLS_DIR="${KOKORO_ARTIFACTS_DIR}/github/gcsfuse-tools"
  PERF_BENCHMARKS_FAILED=0
  
  if [ -d "$TOOLS_DIR" ]; then
    echo "Running Distributed READ Micro-Benchmark from gcsfuse-tools..."
    START_TIME=$SECONDS
    "$TOOLS_DIR/distributed-micro-benchmark/kokoro_run.sh" --commit "$commitId" --read || PERF_BENCHMARKS_FAILED=1
    print_duration "Distributed READ Benchmark" "$START_TIME"
  else
    echo "ERROR: gcsfuse-tools directory not found!"
    PERF_BENCHMARKS_FAILED=1
  fi
  
  if [ $PERF_BENCHMARKS_FAILED -ne 0 ]; then
    echo "Distributed READ benchmarks have failed."
    exit 1
  fi

# =================================================================
# 2) DISTRIBUTED WRITE BENCHMARK
# =================================================================
elif [ "${BENCHMARK_TYPE:-}" == "distributed_benchmark_write" ]; then
  TOOLS_DIR="${KOKORO_ARTIFACTS_DIR}/github/gcsfuse-tools"
  PERF_BENCHMARKS_FAILED=0
  
  if [ -d "$TOOLS_DIR" ]; then
    echo "Running Distributed WRITE Micro-Benchmark from gcsfuse-tools..."
    START_TIME=$SECONDS
    "$TOOLS_DIR/distributed-micro-benchmark/kokoro_run.sh" --commit "$commitId" --write || PERF_BENCHMARKS_FAILED=1
    print_duration "Distributed WRITE Benchmark" "$START_TIME"
  else
    echo "ERROR: gcsfuse-tools directory not found!"
    PERF_BENCHMARKS_FAILED=1
  fi

  if [ $PERF_BENCHMARKS_FAILED -ne 0 ]; then
    echo "Distributed WRITE benchmarks have failed."
    exit 1
  fi

# =================================================================
# 3) LOCAL PERFORMANCE TESTS
# =================================================================
elif [ "${BENCHMARK_TYPE:-}" == "local_tests" ]; then
  # --- Execute local performance tests ---
  echo "Building and installing gcsfuse..."
  BUILD_START=$SECONDS
  ./perfmetrics/scripts/build_and_install_gcsfuse.sh "$commitId"
  print_duration "Build and Install GCSFuse" "$BUILD_START"

  cd "./perfmetrics/scripts/"
  echo "Installing Bigquery module requirements..."
  pip install --require-hashes -r bigquery/requirements.txt --user

  UPLOAD_FLAGS=""
  if [ "${KOKORO_JOB_TYPE:-}" == "RELEASE" ] || \
     [ "${KOKORO_JOB_TYPE:-}" == "CONTINUOUS_INTEGRATION" ] || \
     [ "${KOKORO_JOB_TYPE:-}" == "PRESUBMIT_GITHUB" ] || \
     [ "${KOKORO_JOB_TYPE:-}" == "SUB_JOB" ]; then
    UPLOAD_FLAGS="--upload_gs"
  fi

  COMMON_MOUNT_FLAGS="--log-severity trace --log-format \"text\""

  run_ls_benchmark() {
    local LS_START=$SECONDS
    local ls_flags="$1"
    local spreadsheet_id="$2"
    local config_file="$3"
    local gcsfuse_flags="$COMMON_MOUNT_FLAGS $ls_flags"
    
    echo "Starting LS Benchmark with $config_file..."
    cd "./ls_metrics"
    ./run_ls_benchmark.sh "$gcsfuse_flags" "$UPLOAD_FLAGS" "$spreadsheet_id" "$config_file"
    cd "../"
    print_duration "LS Benchmark ($config_file)" "$LS_START"
  }

  # --- Flat Bucket Tests ---
  echo "Starting Flat Bucket Tests..."
  FLAT_START=$SECONDS
  
  LOG_FILE_LS_TESTS="${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-ls-flat.txt"
  GCSFUSE_LS_FLAGS="--implicit-dirs --client-protocol http1 --log-file $LOG_FILE_LS_TESTS"
  SPREADSHEET_ID='1kvHv1OBCzr9GnFxRu9RTJC7jjQjc9M4rAiDnhyak2Sg'
  LIST_CONFIG_FILE="config.json"
  
  run_ls_benchmark "$GCSFUSE_LS_FLAGS" "$SPREADSHEET_ID" "$LIST_CONFIG_FILE"
  
  print_duration "Flat Bucket Benchmarks (Total)" "$FLAT_START"

  # --- HNS Bucket Tests ---
  echo "Starting HNS Bucket Tests..."
  HNS_START=$SECONDS
  
  SPREADSHEET_ID='1wXRGYyAWvasU8U4KaP7NGPHEvgiOSgMd1sCLxsQUwf0'
  LIST_CONFIG_FILE="config-hns.json"
  
  # 1. HNS with HTTP/1.1
  LOG_FILE_LS_TESTS="${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-ls-hns.txt"
  GCSFUSE_LS_FLAGS_HTTP1="--client-protocol http1 --log-file $LOG_FILE_LS_TESTS"
  run_ls_benchmark "$GCSFUSE_LS_FLAGS_HTTP1" "$SPREADSHEET_ID" "$LIST_CONFIG_FILE"
  
  # 2. HNS with gRPC (default pool size)
  LOG_FILE_LS_TESTS_GRPC="${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-ls-hns-grpc.txt"
  GCSFUSE_LS_FLAGS_GRPC="--client-protocol grpc --log-file $LOG_FILE_LS_TESTS_GRPC"
  run_ls_benchmark "$GCSFUSE_LS_FLAGS_GRPC" "$SPREADSHEET_ID" "$LIST_CONFIG_FILE"
  
  # 3. HNS with gRPC (conn pool size 4)
  LOG_FILE_LS_TESTS_GRPC_POOL4="${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-ls-hns-grpc-pool4.txt"
  GCSFUSE_LS_FLAGS_GRPC_POOL4="--client-protocol grpc --experimental-grpc-conn-pool-size 4 --log-file $LOG_FILE_LS_TESTS_GRPC_POOL4"
  run_ls_benchmark "$GCSFUSE_LS_FLAGS_GRPC_POOL4" "$SPREADSHEET_ID" "$LIST_CONFIG_FILE"
  
  print_duration "HNS Bucket Benchmarks (Total)" "$HNS_START"

  # --- Rename Benchmark ---
  echo "Starting Rename Benchmark..."
  RENAME_START=$SECONDS
  
  cd "./hns_rename_folders_metrics"
  ./run_rename_benchmark.sh $UPLOAD_FLAGS
  cd "../"
  
  print_duration "Rename Benchmark" "$RENAME_START"

# =================================================================
# 4) ZONAL PERFORMANCE TESTS
# =================================================================
elif [ "${BENCHMARK_TYPE:-}" == "distributed_benchmark_zonal" ]; then
  echo "Running Zonal Performance Tests..."
  START_TIME=$SECONDS
  
  # TODO: Add upcoming zonal performance tests.
  echo "Zonal tests scaffolding ready."
  
  print_duration "Zonal Performance Tests" "$START_TIME"

else
  echo "Unknown or unspecified BENCHMARK_TYPE: ${BENCHMARK_TYPE:-}"
  exit 1
fi
