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

# This script will run e2e tests.
# This will stop execution when any command will have non-zero status.
set -e

readonly PROJECT_ID="gcs-fuse-test-ml"
readonly BUCKET_LOCATION=us-central1
readonly REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT="5.3"

# This flag, if set true, will indicate to underlying script(s) to run for zonal bucket(s) instead of non-zonal bucket(s).
ZONAL_FLAG=""
if [[ $# -gt 0 ]]; then
  if [[ "$1" == "true" ]]; then
    ZONAL_FLAG="--zonal"
  elif [[ "$1" != "false" ]]; then
    echo "$0: ZONAL_BUCKET_ARG (\$1) passed as $1. Expected: true or false" >&2
    exit 1
  fi
elif [[ "${RUN_TESTS_WITH_ZONAL_BUCKET}" == "true" ]]; then
  echo "Running for zonal bucket(s) ..."
  ZONAL_FLAG="--zonal"
fi

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"

# Install required bash version for e2e script as kokoro has outdated bash versions.
./perfmetrics/scripts/install_bash.sh "$REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT"

echo "Building and installing gcsfuse..."
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
./perfmetrics/scripts/build_and_install_gcsfuse.sh $commitId

# To execute tests for a specific commitId, ensure you've checked out from that commitId first.
git checkout $commitId

echo "Running e2e tests on installed package...."
/usr/local/bin/bash ./tools/integration_tests/improved_run_e2e_tests.sh --bucket-location=$BUCKET_LOCATION --test-installed-package ${ZONAL_FLAG}
