#!/bin/bash
# Copyright 2023 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     [http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e
set -x

# 0. Environment Setup
# ------------------------
# Set default benchmark type if not provided
BENCHMARK_TYPE=${BENCHMARK_TYPE:-"periodic"}

# Default to test project if not set (Safe for local debug)
PROJECT_ID=${PROJECT_ID:-"gcs-fuse-test"}

# KOKORO FLAG LOGIC
KOKORO_FLAG=""
IS_KOKORO=false
if [ -n "$KOKORO_BUILD_ID" ]; then
  KOKORO_FLAG="--kokoro"
  IS_KOKORO=true
fi

# Ensure PROJECT_ID is set (Required for both Local and Kokoro)
if [ -z "$PROJECT_ID" ]; then
  echo "Error: PROJECT_ID is not set."
  echo "Usage (Local): PROJECT_ID=your-project-id ./build.sh"
  exit 1
fi

echo "Setting up environment (Kokoro: $IS_KOKORO)..."

# Only install system packages if running in Kokoro (avoid sudo locally)
# if [ "$IS_KOKORO" = true ]; then
sudo apt-get update && sudo apt-get install -y python3-venv python3-pip git
# fi

# Verify Python version (Tool requires 3.12+, but we log what we have)
python3 --version

# Capture the absolute path to the gcsfuse repo root (assuming script runs from root)
# This allows us to reference the upload script later even after changing directories.
GCSFUSE_REPO_ROOT=$(pwd)

# Create a temporary workspace to avoid polluting the gcsfuse repo with the tools clone
WORKSPACE_DIR=$(mktemp -d)
echo "Setting up workspace at $WORKSPACE_DIR"
cd "$WORKSPACE_DIR"

# 1. Clone & Prep
echo "Cloning gcsfuse-tools..."
git clone https://github.com/GoogleCloudPlatform/gcsfuse-tools.git
cd gcsfuse-tools/gcsfuse-micro-benchmarking
git checkout microbenchmark-gce-vm-ssh-compatibility

# 2. Virtual Env
echo "Creating Python virtual environment..."
python3 -m venv venv || { echo "Venv creation failed. Check if python3-venv is installed."; exit 1; }
source venv/bin/activate
pip install -r requirements.txt --extra-index-url https://pypi.org/simple
# Force pip to check the public index for BigQuery
pip install google-cloud-bigquery --extra-index-url https://pypi.org/simple

# 3. Auth & SSH
# The python tool expects to be able to SSH into the new VM.
# We must generate an SSH key and add it to the local agent.
echo "Configuring SSH..."
mkdir -p ~/.ssh
# Generate a fresh keypair for this run (using a specific file to avoid overwriting default id_rsa)
ssh-keygen -t rsa -f ~/.ssh/google_compute_engine -N "" -C "kokoro-bot" -q <<< y >/dev/null 2>&1 || true

# Add key to ssh-agent
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/google_compute_engine

# Ensure gcloud uses the Service Account credentials
if [ -n "$GOOGLE_APPLICATION_CREDENTIALS" ]; then
  gcloud auth activate-service-account --key-file="$GOOGLE_APPLICATION_CREDENTIALS"
fi
gcloud config set project "$PROJECT_ID"

# 4. Config Creation
# We generate the config dynamically to inject specific Kokoro variables.
# We use the 'resources/' directory which contains the default templates in the repo.
cat <<EOF > kokoro_bench_config.yaml
zonal_benchmarking: False
reuse_same_mount: True
iterations: 2
bench_env:
  delete_after_use: True
  zone: "us-west1-b"
  project: "$PROJECT_ID"
  gce_env:
    # Empty vm_name triggers creation of a new VM
    vm_name: ""
    machine_type: "n2-standard-16"
    image_family: "debian-11"
    image_project: "debian-cloud"
    disk_size: 100
  gcs_bucket:
    bucket_name: "" # Empty triggers creation of a new bucket
fio_jobfile_template: "resources/jobfile.fio"
job_details:
  file_path: "resources/fio_job_cases.csv"
version_details:
  gcsfuse_version_or_commit: "master"
EOF

# 5. Run the Benchmark Tool
# We use the current timestamp as the ID to avoid collisions
BENCH_ID="cpranjal-trial-kokoro-run-$(date +%Y%m%d-%H%M)"

echo "Starting Benchmark ID: $BENCH_ID"
python3 main.py \
  --benchmark_id_prefix="$BENCH_ID" \
  --config_filepath="kokoro_bench_config.yaml" \
  --bench_type="$BENCHMARK_TYPE"

# 6. Upload Results to BigQuery
echo "Starting BigQuery Upload..."

# 6a. Define the GCS path.
GCS_RESULT_PATH="gs://gcsfuse-perf-benchmark-artifacts/${BENCH_ID}*/result.json"
LOCAL_RESULT="/tmp/result_to_upload.json"

echo "Fetching result from GCS: $GCS_RESULT_PATH"

# 6b. Download the file from GCS to a safe local spot
if gcloud storage cp "$GCS_RESULT_PATH" "$LOCAL_RESULT"; then
    echo "Result file downloaded successfully."
    
    # 6c. Run the uploader using the local copy
    # We use the captured GCSFUSE_REPO_ROOT because we are still inside the temporary workspace
    UPLOAD_OUT=$(python3 "$GCSFUSE_REPO_ROOT/perfmetrics/scripts/continuous_test/gcp_ubuntu/upload_to_bq.py" \
        --result_file="$LOCAL_RESULT" $KOKORO_FLAG)
    
    TABLE_ID=$(echo "$UPLOAD_OUT" | grep "RESULT_TABLE_ID=" | cut -d'=' -f2)

    echo "-----------------------------------------------------------"
    echo "BENCHMARK COMPLETE"
    if [ "$IS_KOKORO" = true ]; then
        echo "View Periodic Metrics: http://plx/dashboards/periodic_kokoro?table_id=$TABLE_ID"
    else
        echo "View Local Metrics: http://plx/dashboards/xyz?table_id=$TABLE_ID"
    fi
    echo "-----------------------------------------------------------"
else
    echo "Error: Could not find result.json at $GCS_RESULT_PATH."
fi

echo "Kokoro Benchmark Completed Successfully."
