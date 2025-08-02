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

set -e

VM_NAME="gcsfuse-micro-benchmark-tests"
ZONE="us-west1-b"
SCRIPT_LOCAL_PATH="./perfmetrics/scripts/micro_benchmarks/run_microbenchmark.sh"
SCRIPT_REMOTE_PATH="/tmp/run_microbenchmark.sh"
q
# --- Upload script to VM ---
echo "Copying script to VM..."
gcloud compute scp "$SCRIPT_LOCAL_PATH" "$VM_NAME:$SCRIPT_REMOTE_PATH" --zone "$ZONE"

# --- Run script on VM ---
echo "Running script on VM..."
gcloud compute ssh "$VM_NAME" --zone "$ZONE" --command "bash $SCRIPT_REMOTE_PATH"

echo "Script executed successfully on VM."
