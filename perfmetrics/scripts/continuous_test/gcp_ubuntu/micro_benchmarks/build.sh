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

set -euo pipefail

VM_NAME="periodic-micro-benchmark-tests"
ZONE="us-west1-b"
REPO_DIR="~/github/gcsfuse"
MOUNTED_DIR="$REPO_DIR/perfmetrics/scripts/micro_benchmarks/gcs"
TEST_SCRIPT_PATH="github/gcsfuse/perfmetrics/scripts/micro_benchmarks/run_microbenchmark.sh"
GCSFUSE_REPO="https://github.com/GoogleCloudPlatform/gcsfuse.git"

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

run_script_on_vm() {
  log "Running benchmark script on VM with clean setup..."

  sudo gcloud compute ssh "$VM_NAME" --zone "$ZONE" --internal-ip --command "
    set -euxo pipefail

    MOUNTED_DIR=\"$MOUNTED_DIR\"
    GCSFUSE_REPO=\"$GCSFUSE_REPO\"
    TEST_SCRIPT_PATH=\"$TEST_SCRIPT_PATH\"

    sudo apt-get update -y
    sudo apt-get install -y git

    # Unmount if gcsfuse mount exists
    if mountpoint -q \"\$MOUNTED_DIR\"; then
      echo \"\$MOUNTED_DIR is mounted. Attempting to unmount...\"
      sudo fusermount -u \"\$MOUNTED_DIR\" || sudo umount \"\$MOUNTED_DIR\"
    fi

    # Clean up any existing repo
    rm -rf ~/github

    # Clone fresh repo
    mkdir -p ~/github
    git clone \"\$GCSFUSE_REPO\" ~/github/gcsfuse
    cd ~/github/gcsfuse
    commitId=\$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
    git checkout \$commitId

    # Run benchmark
    echo \"Triggering benchmark script...\"
    bash ~/\$TEST_SCRIPT_PATH
  "

  log "Benchmark script executed successfully on VM."
}

# ---- Main Execution ----
run_script_on_vm
