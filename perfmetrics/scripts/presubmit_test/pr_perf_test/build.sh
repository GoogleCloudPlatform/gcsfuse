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

# This will stop execution when any command will have non-zero status.

set -e
sudo apt-get update

# e.g. architecture=arm64 or amd64
echo "Installing git"
sudo apt-get install git
cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"

#echo "Building and installing gcsfuse..."
#chmod +x perfmetrics/scripts/build_and_install_package.sh
#./perfmetrics/scripts/build_and_install_package.sh
LOG_FILE=${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs.txt
echo "Running e2e tests...."
chmod +x perfmetrics/scripts/run_e2e_tests.sh
./perfmetrics/scripts/run_e2e_tests.sh false
