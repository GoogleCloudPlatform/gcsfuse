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
readonly SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE=true
readonly RUN_TEST_ON_TPC_ENDPOINT=false
readonly PROJECT_ID="gcs-fuse-test-ml"
readonly BUCKET_LOCATION=us-central1

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
echo "Building and installing gcsfuse..."
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
./perfmetrics/scripts/build_and_install_gcsfuse.sh $commitId

# To execute tests for a specific commitId, ensure you've checked out from that commitId first.
git checkout $commitId

echo "Running e2e tests on installed package...."
# $1 argument is refering to value of testInstalledPackage
./tools/integration_tests/run_e2e_tests.sh $RUN_E2E_TESTS_ON_INSTALLED_PACKAGE $SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE $BUCKET_LOCATION $RUN_TEST_ON_TPC_ENDPOINT
./tools/integration_tests/run_e2e_tests.sh $RUN_E2E_TESTS_ON_INSTALLED_PACKAGE $SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE $RUN_TEST_ON_TPC_ENDPOINT
