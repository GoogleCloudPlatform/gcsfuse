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
apt-get update && apt-get install -y wget git build-essential ca-certificates sudo

echo "Step 2: Cloning GCSFuse repo..."
git clone -b "$GCSFUSE_BRANCH" https://github.com/GoogleCloudPlatform/gcsfuse.git
cd gcsfuse

GO_VERSION=$(cat .go-version | tr -d '[:space:]')
if [[ ! "$GO_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Error: Invalid Go version format in .go-version"
    exit 1
fi
echo "Step 3: Installing Go..."
./perfmetrics/scripts/install_go.sh "$GO_VERSION"
export PATH="/usr/local/go/bin:$PATH"
echo "Go version installed:"
go version

echo "Step 4: Running tests ..."
# These tests are chosen to verify that machine-type is correctly passed by
# CSI Driver to GCSFuse and GCSFuse is correctly accepting it and triggering optimization flags
# like implicit-dirs and rename-dir-limit for high-performance machine-type as expected.
go test -v ./tools/integration_tests/flag_optimizations/... --integrationTest --mountedDirectory=/data_mnt --testbucket="$BUCKET_NAME" -run "TestImplicitDirsEnabled|TestRenameDirLimitSet"

echo "Step 5: Test finished successfully."
