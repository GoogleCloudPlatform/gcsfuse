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

# Get the latest commitId of yesterday in the log file.
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)

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
    # Exit successfully so it doesn't run the rest of the script
  exit 0

# =================================================================
# 2) DISTRIBUTED WRITE BENCHMARK + LOCAL PERF TESTS
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

  COMMON_MOUNT_FLAGS="--debug_fuse --debug_gcs --log-format \"text\""

  run_load_test_and_fetch_metrics() {
    local FIO_START=$SECONDS
    local fio_flags="$1"
    local bucket_name="$2"
    local spreadsheet_id="$3"
    local gcsfuse_flags="$COMMON_MOUNT_FLAGS $fio_flags"
    
    echo "Starting FIO Load Test on $bucket_name..."
    ./run_load_test_and_fetch_metrics.sh "$gcsfuse_flags" "$UPLOAD_FLAGS" "$bucket_name" "$spreadsheet_id"
    print_duration "FIO Load Test ($bucket_name)" "$FIO_START"
  }

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
  
  LOG_FILE_FIO_TESTS="${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-fio-flat.txt"
  LOG_FILE_LS_TESTS="${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-ls-flat.txt"
  GCSFUSE_FIO_FLAGS="--implicit-dirs --stackdriver-export-interval=30s --log-file $LOG_FILE_FIO_TESTS"
  GCSFUSE_LS_FLAGS="--implicit-dirs --log-file $LOG_FILE_LS_TESTS"
  BUCKET_NAME="periodic-perf-tests"
  SPREADSHEET_ID='1kvHv1OBCzr9GnFxRu9RTJC7jjQjc9M4rAiDnhyak2Sg'
  LIST_CONFIG_FILE="config.json"
  
  run_load_test_and_fetch_metrics "$GCSFUSE_FIO_FLAGS" "$BUCKET_NAME" "$SPREADSHEET_ID"
  run_ls_benchmark "$GCSFUSE_LS_FLAGS" "$SPREADSHEET_ID" "$LIST_CONFIG_FILE"
  
  print_duration "Flat Bucket Benchmarks (Total)" "$FLAT_START"

  # --- HNS Bucket Tests ---
  echo "Starting HNS Bucket Tests..."
  HNS_START=$SECONDS
  
  LOG_FILE_FIO_TESTS="${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-fio-hns.txt"
  LOG_FILE_LS_TESTS="${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-ls-hns.txt"
  GCSFUSE_FIO_FLAGS="--stackdriver-export-interval=30s --log-file $LOG_FILE_FIO_TESTS"
  GCSFUSE_LS_FLAGS="--log-file $LOG_FILE_LS_TESTS"
  BUCKET_NAME="periodic-perf-tests-hns"
  SPREADSHEET_ID='1wXRGYyAWvasU8U4KaP7NGPHEvgiOSgMd1sCLxsQUwf0'
  LIST_CONFIG_FILE="config-hns.json"
  
  run_load_test_and_fetch_metrics "$GCSFUSE_FIO_FLAGS" "$BUCKET_NAME" "$SPREADSHEET_ID"
  run_ls_benchmark "$GCSFUSE_LS_FLAGS" "$SPREADSHEET_ID" "$LIST_CONFIG_FILE"
  
  print_duration "HNS Bucket Benchmarks (Total)" "$HNS_START"

  # --- Rename Benchmark ---
  echo "Starting Rename Benchmark..."
  RENAME_START=$SECONDS
  
  cd "./hns_rename_folders_metrics"
  ./run_rename_benchmark.sh $UPLOAD_FLAGS
  
  print_duration "Rename Benchmark" "$RENAME_START"
  exit 0

# =================================================================
# 3) ZONAL PERFORMANCE TESTS
# =================================================================
elif [ "${BENCHMARK_TYPE:-}" == "distributed_benchmark_zonal" ]; then
  echo "Running Zonal Performance Tests..."
  START_TIME=$SECONDS
  
  # TODO: Add upcoming zonal performance tests.
  echo "Zonal tests scaffolding ready."
  
  print_duration "Zonal Performance Tests" "$START_TIME"
  exit 0
  
else
  echo "Unknown or unspecified BENCHMARK_TYPE: ${BENCHMARK_TYPE:-}"
  exit 1
fi
# TODO: Testing for hns bucket with client protocol set to grpc. To be done when
#  includeFolderAsPrefixes is supported in grpc.
# TODO: Testing for hns bucket with client protocol set to grpc with grpc-conn-pool-size
# set to 4. To be done when includeFolderAsPrefixes is supported in grpc.
