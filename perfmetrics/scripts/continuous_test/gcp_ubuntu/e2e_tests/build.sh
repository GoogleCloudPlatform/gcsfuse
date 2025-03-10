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

readonly RUN_E2E_TESTS_ON_INSTALLED_PACKAGE=true
readonly SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE=false
readonly RUN_TEST_ON_TPC_ENDPOINT=false
readonly PROJECT_ID="gcs-fuse-test-ml"
readonly BUCKET_LOCATION=us-central1

# This flag, if set true, will indicate to the underlying script, to customize for a presubmit-run.
readonly RUN_TESTS_WITH_PRESUBMIT_FLAG=false

# This flag, if set true, will indicate to underlying script(s) to run for zonal bucket(s) instead of non-zonal bucket(s).
ZONAL_BUCKET_ARG=false
if [ $# -gt 0 ]; then
  if [ $1 = "true" ]; then
    ZONAL_BUCKET_ARG=true
  elif [ $1 != "false" ]; then
    >&2 echo "$0: ZONAL_BUCKET_ARG (\$1) passed as $1 . Expected: true or false"
    exit  1
  fi
elif test -n ${RUN_TESTS_FOR_ZONAL_BUCKET}; then
  ZONAL_BUCKET_ARG=${RUN_TESTS_FOR_ZONAL_BUCKET}
fi

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
echo "Building and installing gcsfuse..."
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
./perfmetrics/scripts/build_and_install_gcsfuse.sh $commitId

# To execute tests for a specific commitId, ensure you've checked out from that commitId first.
git checkout $commitId

echo "Running e2e tests on installed package...."
# $1 argument is refering to value of testInstalledPackage
./tools/integration_tests/run_e2e_tests.sh $RUN_E2E_TESTS_ON_INSTALLED_PACKAGE $SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE $BUCKET_LOCATION $RUN_TEST_ON_TPC_ENDPOINT $RUN_TESTS_WITH_PRESUBMIT_FLAG ${ZONAL_BUCKET_ARG}
