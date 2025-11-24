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

# Fail on any error.
set -e
set -x
echo "Running JAX checkpoint tests"

sudo apt-get update
# Install Git.
echo "Installing git"
sudo apt-get install git
# Install Golang.
#wget -O go_tar.tar.gz https://go.dev/dl/go1.24.9.linux-amd64.tar.gz -q
architecture=$(dpkg --print-architecture)
wget -O go_tar.tar.gz https://go.dev/dl/go1.24.9.linux-${architecture}.tar.gz -q
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go_tar.tar.gz
export PATH=$PATH:/usr/local/go/bin
# Install latest gcloud version for compatability with HNS bucket.
function upgrade_gcloud_version() {
  sudo apt-get update
  gcloud version
  wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
  sudo tar xzf gcloud.tar.gz && sudo cp -r google-cloud-sdk /usr/local && sudo rm -r google-cloud-sdk
  sudo /usr/local/google-cloud-sdk/install.sh
  export PATH=/usr/local/google-cloud-sdk/bin:$PATH
  echo 'export PATH=/usr/local/google-cloud-sdk/bin:$PATH' >> ~/.bashrc
  gcloud version && rm gcloud.tar.gz
  sudo /usr/local/google-cloud-sdk/bin/gcloud components update
  sudo /usr/local/google-cloud-sdk/bin/gcloud components install alpha
}
upgrade_gcloud_version

# Build gcsfuse.
cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
CGO_ENABLED=0 go build .

function mount_gcsfuse_and_run_test() {
  # Function to mount GCSFuse.
  # Input:
  #   $1: Bucket name

  local BUCKET_NAME="$1"
  # Clean up bucket before run (ignoring the failure if there are no objects to delete).
  gcloud alpha storage rm -r gs://${BUCKET_NAME}/**  || true
  # Create a directory for gcsfuse logs.
  mkdir -p "${KOKORO_ARTIFACTS_DIR}/gcsfuse_logs"
  local MOUNT_POINT="${HOME}/gcs/${BUCKET_NAME}"
  mkdir -p "${MOUNT_POINT}"

  COMMON_FLAGS=(--log-severity=TRACE --enable-streaming-writes --log-file="${KOKORO_ARTIFACTS_DIR}"/gcsfuse_logs/"${BUCKET_NAME}".log)
  if [[ "$BUCKET_NAME" =~ "flat" ]]; then
    go run . "${COMMON_FLAGS[@]}" --rename-dir-limit=100  "${BUCKET_NAME}" "${MOUNT_POINT}"
  else
    go run . "${COMMON_FLAGS[@]}" "${BUCKET_NAME}" "${MOUNT_POINT}"
  fi
  python3.11 ./perfmetrics/scripts/ml_tests/checkpoint/Jax/emulated_checkpoints.py --checkpoint_dir "${MOUNT_POINT}"
}

# Enable python virtual environment.
# By default KOKORO VM installs python 3.8 which causes dependency issues.
# Following commands are to explicitly set up python 3.11.
sudo apt update
sudo apt install -y software-properties-common
sudo add-apt-repository -y ppa:deadsnakes/ppa
sudo apt update
sudo apt install -y python3.11 python3.11-dev python3.11-venv
# Install pip
curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py
python3.11 get-pip.py
rm get-pip.py
python3.11 -m venv .venv
source .venv/bin/activate
# Install JAX dependencies.
pip install -r ./perfmetrics/scripts/ml_tests/checkpoint/Jax/requirements.txt

ZONE=$(curl -s -H "Metadata-Flavor: Google" http://metadata/computeMetadata/v1/instance/zone | cut -d'/' -f4)
# Run tests in parallel on flat, hns and zonal bucket.
FLAT_BUCKET_NAME="jax-emulated-checkpoint-flat-${architecture}"
HNS_BUCKET_NAME="jax-emulated-checkpoint-hns-${architecture}"
ZONAL_BUCKET_NAME="jax-emulated-checkpoint-zonal-${ZONE}-${architecture}"
mount_gcsfuse_and_run_test "${FLAT_BUCKET_NAME}" &
flat_pid=$!
mount_gcsfuse_and_run_test "${HNS_BUCKET_NAME}" &
hns_pid=$!
mount_gcsfuse_and_run_test "${ZONAL_BUCKET_NAME}" &
zonal_pid=$!

# Wait for all processes to finish and check exit codes
wait "$flat_pid"
flat_status=$?
wait "$hns_pid"
hns_status=$?
wait "$zonal_pid"
zonal_status=$?

if [[ "$flat_status" -ne 0 ]] || [[ "$hns_status" -ne 0 ]] || [[ "$zonal_status" -ne 0 ]]; then
  echo "Checkpoint tests failed"
  exit 1
else
  echo "Checkpoint tests completed successfully"
fi
