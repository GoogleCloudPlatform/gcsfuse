#!/bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e
sudo apt-get update

echo "Installing git"
sudo apt-get install git
echo "Installing go-lang 1.21.0"
wget -O go_tar.tar.gz https://go.dev/dl/go1.21.0.linux-amd64.tar.gz -q
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"

# Checkout back to master branch to use latest CI test scripts in master.
git checkout master

echo "Building and installing gcsfuse"
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
chmod +x perfmetrics/scripts/build_and_install_gcsfuse.sh
./perfmetrics/scripts/build_and_install_gcsfuse.sh $commitId

# Mounting gcs bucket
cd "./perfmetrics/scripts/"
echo "Mounting gcs bucket"
mkdir -p gcs
LOG_FILE=${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs.txt
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --debug_fuse --debug_gcs --log-file $LOG_FILE --log-format \"text\" --stackdriver-export-interval=30s"
BUCKET_NAME=periodic-perf-tests
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
gcsfuse $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT

# Executing perf tests
chmod +x run_load_test_and_fetch_metrics.sh
./run_load_test_and_fetch_metrics.sh

sudo umount $MOUNT_POINT

# ls_metrics test. This test does gcsfuse mount first and then do the testing.
cd "./ls_metrics"
chmod +x run_ls_benchmark.sh
./run_ls_benchmark.sh
