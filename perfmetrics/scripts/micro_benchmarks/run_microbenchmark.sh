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

set -euo pipefail  # Exit on error, unset variables are errors, pipe fails propagate

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

# --- Constants ---
VENV_DIR="venv"
ARTIFACT_BUCKET_PATH="gcsfuse-kokoro-logs/prod/gcsfuse/gcp_ubuntu/periodic/micro_benchmark"
DATE=$(date +%Y-%m-%d)

# --- Functions ---
cleanup_mounts() {
  log "Cleaning up any stale gcsfuse mounts..."
  for mnt in $(mount | grep gcsfuse | awk '{print $3}'); do
    log "Unmounting $mnt"
    sudo fusermount -u "$mnt" || true
  done
}


prepare_venv() {
  log "Setting up Python virtual environment..."
  if [[ ! -d "$VENV_DIR" ]]; then
    python3 -m venv "$VENV_DIR"
  fi
  source "$VENV_DIR/bin/activate"
  pip install --require-hashes -r "requirements.txt" --user
}

run_benchmark() {
  local rw=$1                # "read" or "write" operation type
  local script_path=$2       # Path to the benchmark script (e.g., read_single_thread.py)
  local file_size_gb=$3      # Size of each file to read/write in GB
  local total_files=$4       # Total number of files to process

  echo "Running $rw benchmark with file size $file_size_gb GB and total files $total_files..."
  local log_file="/tmp/gcsfuse-logs-single-threaded-${rw}-${file_size_gb}gb-test.txt"

  # Clean old log file if it exists
  rm -f "$log_file"

  # Pass log file flag as a string.
  local gcsfuse_flags="--log-file $log_file"

  log "Running $rw benchmark..."
  if ! python3 "$script_path" --bucket single-threaded-tests \
      --gcsfuse-config "$gcsfuse_flags" \
      --total-files "$total_files" \
      --file-size-gb "$file_size_gb"; then
    log "$rw benchmark failed. Copying log to gs://$ARTIFACT_BUCKET_PATH/$DATE"
    gcloud storage cp "$log_file" "gs://$ARTIFACT_BUCKET_PATH/$DATE/"
    gcloud storage cat "gs://$ARTIFACT_BUCKET_PATH/$DATE/$(basename "$log_file")"
    return 1
  fi

  return 0
}

# --- Main Script ---
log "Installing dependencies..."
sudo apt-get update -y
sudo apt-get install -y git gnupg python3-venv

cd "$HOME/github/gcsfuse"
commitId=$(git rev-parse --short HEAD)
./perfmetrics/scripts/build_and_install_gcsfuse.sh $commitId

cd "perfmetrics/scripts/micro_benchmarks"
# Cleanup previous mounts if any
cleanup_mounts
prepare_venv

READ_GB=15
TOTAL_READ_FILES=10
WRITE_GB=15
TOTAL_WRITE_FILES=1
exit_code=0

if ! run_benchmark "read" "read_single_thread.py" "$READ_GB" "$TOTAL_READ_FILES"; then
  echo "Read benchmark failed."
  exit_code=1
fi

if ! run_benchmark "write" "write_single_thread.py" "$WRITE_GB" "$TOTAL_WRITE_FILES"; then
  echo "Write benchmark failed."
  exit_code=1
fi

deactivate || true
cleanup_mounts

if [[ $exit_code -ne 0 ]]; then
  log "One or more benchmarks failed."
  exit $exit_code
fi

log "Benchmarks completed successfully."
exit 0
