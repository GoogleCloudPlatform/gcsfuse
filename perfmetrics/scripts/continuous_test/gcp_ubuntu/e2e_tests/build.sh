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

# This script will run e2e tests.
# This will stop execution when any command will have non-zero status.
set -e

readonly RUN_E2E_TESTS_ON_INSTALLED_PACKAGE=true

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
echo "Building and installing gcsfuse..."
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
commitId=$(git log origin/$BRANCH_NAME --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
./perfmetrics/scripts/build_and_install_gcsfuse.sh $commitId

echo "Running e2e tests on installed package...."
# $1 argument is refering to value of testInstalledPackage
./perfmetrics/scripts/run_e2e_tests.sh $RUN_E2E_TESTS_ON_INSTALLED_PACKAGE
