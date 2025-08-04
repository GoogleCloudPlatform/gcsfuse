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

#!/bin/bash

set -euo pipefail

VM_NAME="periodic-micro-benchmark-tests"
ZONE="us-west1-b"
TEST_SCRIPT_PATH="github/gcsfuse/perfmetrics/scripts/micro_benchmarks/run_microbenchmark.sh"
GCSFUSE_REPO="https://github.com/GoogleCloudPlatform/gcsfuse.git"
REPO_DIR="github/gcsfuse"

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

initialize_ssh_key() {
  log "Cleaning up old OS Login SSH keys..."

  local existing_keys
  existing_keys=$(gcloud compute os-login ssh-keys list --format="value(key)" || true)

  if [[ -n "$existing_keys" ]]; then
    while IFS= read -r key; do
      gcloud compute os-login ssh-keys remove --key="$key"
    done <<< "$existing_keys"
  else
    log "No SSH keys to remove."
  fi

  log "Initializing SSH access to VM: $VM_NAME..."

  local delay=1 max_delay=10 attempt=1 max_attempts=5

  while (( attempt <= max_attempts )); do
    log "SSH connection attempt $attempt..."
    if gcloud compute ssh "$VM_NAME" --zone "$ZONE" --internal-ip --quiet --command "echo 'SSH OK on $VM_NAME'" &>/dev/null; then
      log "SSH connection established."
      return 0
    fi
    log "SSH connection failed. Retrying in ${delay}s..."
    sleep "$delay"
    delay=$((delay * 2))
    (( delay > max_delay )) && delay=$max_delay
    attempt=$((attempt + 1))
  done

  log "ERROR: All SSH connection attempts failed."
  return 1
}

run_script_on_vm() {
  log "Running clean setup and benchmark script on VM..."

  gcloud compute ssh "$VM_NAME" --zone "$ZONE" --internal-ip --command "
    set -euxo pipefail

    # Ensure clean environment
    sudo apt-get update -y
    sudo apt-get install -y git

    rm -rf ~/github
    mkdir -p ~/github

    git clone $GCSFUSE_REPO ~/github/gcsfuse
    cd ~/github/gcsfuse
    git checkout master
    git pull origin master

    echo 'Triggering benchmark script...'
    bash ~/$TEST_SCRIPT_PATH
  "

  log "Benchmark script executed successfully on VM."
}

# ---- Main Execution ----
initialize_ssh_key
run_script_on_vm
