#!/bin/bash
# Copyright 2024 Google LLC
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

# This script will run e2e tests for tpc.
# This will stop execution when any command will have non-zero status.
set -e

readonly RUN_E2E_TESTS_ON_INSTALLED_PACKAGE=true
readonly SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE=true
readonly RUN_TEST_ON_TPC_ENDPOINT=true
# TPC project id
readonly PROJECT_ID="tpczero-system:gcsfuse-test-project"
readonly BUCKET_LOCATION="u-us-prp1"
readonly REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT="5.3"

# This flag, if set true, will indicate to underlying script to customize for a presubmit run.
readonly RUN_TESTS_WITH_PRESUBMIT_FLAG=false

# This flag, if set true, will indicate to underlying script to also run for zonal buckets.
readonly RUN_TESTS_WITH_ZONAL_BUCKET=false

# Install required bash version for e2e script as kokoro has outdated bash versions.
./perfmetrics/scripts/install_bash.sh "$REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT"

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"

# Install required bash version for e2e script as kokoro has outdated bash versions.
./perfmetrics/scripts/install_bash.sh "$REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT"

# Install latest gcloud.
./perfmetrics/scripts/install_latest_gcloud.sh
export PATH="/usr/local/google-cloud-sdk/bin:$PATH"

# Copy the key file for the TPC service account to use for authentication.
gcloud storage cp gs://gcsfuse-tpc-tests/creds.json /tmp/sa.key.json

echo "Building and installing gcsfuse..."
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
./perfmetrics/scripts/build_and_install_gcsfuse.sh $commitId

## To execute tests for a specific commitId, ensure you've checked out that commitId first.
git checkout $commitId
echo "Running e2e tests on installed package...."

# Initiate PRPTST environment to establish a TPC project and associated account.
gcloud config configurations create prptst
gcloud config configurations activate prptst
gcloud config set universe_domain apis-tpczero.goog
gcloud config set api_endpoint_overrides/compute https://compute.apis-tpczero.goog/compute/v1/
gcloud auth activate-service-account --key-file=/tmp/sa.key.json
gcloud config set project $PROJECT_ID

set +e
# $1 argument is refering to value of testInstalledPackage
/usr/local/bin/bash ./tools/integration_tests/improved_run_e2e_tests.sh --bucket-location=$BUCKET_LOCATION --test-installed-package=true --skip-non-essential-tests=true --test-on-tpc-endpoint=true --presubmit=$RUN_TESTS_WITH_PRESUBMIT_FLAG --zonal=${RUN_TESTS_WITH_ZONAL_BUCKET}
exit_code=$?
set -e

# Activate default environment after testing.
gcloud config configurations activate default
gcloud config unset universe_domain
gcloud config unset api_endpoint_overrides/compute
gcloud config unset project
exit $exit_code
