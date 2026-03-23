#!/bin/bash
# Copyright 2023 Google LLC
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

# Script to run e2e tests for regional or zonal buckets if env variable RUN_TESTS_WITH_ZONAL_BUCKET is set to 'true'.
# Exit on error, treat unset variables as errors, and propagate pipeline errors.
set -euo pipefail

if [[ $# -gt 0 ]]; then
    echo "This script requires no argument. Pass env variable RUN_TESTS_WITH_ZONAL_BUCKET set to 'true' to run this script for zonal buckets." 
    exit 1
fi

readonly BUCKET_LOCATION="us-central1"

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"

echo "Building and installing gcsfuse..."
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
./perfmetrics/scripts/build_and_install_gcsfuse.sh $commitId

# To execute tests for a specific commitId, ensure you've checked out from that commitId first.
git checkout $commitId

if [[ "${RUN_TESTS_WITH_ZONAL_BUCKET-}" == "true" ]]; then
    echo "Running zonal e2e tests on installed package...."
    bash ./tools/integration_tests/improved_run_e2e_tests.sh --bucket-location="$BUCKET_LOCATION" --test-installed-package --zonal
else
    if [[ -n "${RUN_TESTS_WITH_ZONAL_BUCKET-}" ]]; then
        echo "Warning: RUN_TESTS_WITH_ZONAL_BUCKET is set to '${RUN_TESTS_WITH_ZONAL_BUCKET}', which is not 'true'. Running regional tests."
    else
        echo "RUN_TESTS_WITH_ZONAL_BUCKET is not set. Running regional tests by default."
    fi
    echo "Running regional e2e tests on installed package...."
    bash ./tools/integration_tests/improved_run_e2e_tests.sh --bucket-location="$BUCKET_LOCATION" --test-installed-package
fi
