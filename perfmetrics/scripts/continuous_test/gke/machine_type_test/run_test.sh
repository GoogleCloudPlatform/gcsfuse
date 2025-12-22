#!/bin/bash
# Copyright 2026 Google LLC
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
echo "Step 1: Container started. Updating apt and installing dependencies..."
apt-get update && apt-get install -y wget git build-essential ca-certificates

echo "Step 2: Installing Go 1.24.11 ..."
wget -q https://go.dev/dl/go1.24.11.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.24.11.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo "Go version installed:"
go version

echo "Step 3: Cloning GCSFuse repo..."
git clone -b "$GCSFUSE_BRANCH" https://github.com/GoogleCloudPlatform/gcsfuse.git
cd gcsfuse
echo "Repo content:"
ls -F

echo "Step 4: Running tests ..."
# These tests are chosen to verify that machine-type is correctly passed by
# CSI Driver to GCSFuse and GCSFuse is correctly accepting it and triggering optimization flags
# like implicit-dirs and rename-dir-limit for high-performance machine-type as expected.
go test -v ./tools/integration_tests/flag_optimizations/... --integrationTest --mountedDirectory=/data_mnt --testbucket="$BUCKET_NAME" -run "TestImplicitDirsEnabled|TestRenameDirLimitSet"

echo "Step 5: Test finished successfully."
