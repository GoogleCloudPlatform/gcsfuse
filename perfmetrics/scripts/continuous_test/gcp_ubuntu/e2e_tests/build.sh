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

# Script to run e2e tests for regional or zonal buckets.
# Exit on error, treat unset variables as errors, and propagate pipeline errors.
set -euo pipefail

if [[ $# -gt 1 ]]; then
    echo "This script requires at most one argument" 
    echo "Usage: $0 true, for zonal e2e test runs."
    echo "Usage: $0 false, for regional e2e test runs."
    echo "Example: $0 false"
    exit 1
fi

readonly BUCKET_LOCATION="us-central1"
readonly REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT="5.1"
readonly INSTALL_BASH_VERSION="5.3" # Using 5.3 for installation as bash 5.1 has an installation bug.
IS_ZONAL=${1:-false}

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"

# Install latest gcloud version.
./perfmetrics/scripts/install_latest_gcloud.sh
export PATH="/usr/local/google-cloud-sdk/bin:$PATH"

echo "Building and installing gcsfuse..."
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
./perfmetrics/scripts/build_and_install_gcsfuse.sh $commitId

# To execute tests for a specific commitId, ensure you've checked out from that commitId first.
git checkout $commitId

# Check and install required bash version for e2e script.
BASH_EXECUTABLE="bash"
REQUIRED_BASH_MAJOR=$(echo "$REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT" | cut -d'.' -f1)
REQUIRED_BASH_MINOR=$(echo "$REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT" | cut -d'.' -f2)

echo "Current Bash version: ${BASH_VERSINFO[0]}.${BASH_VERSINFO[1]}"
echo "Required Bash version for e2e script: ${REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT}"

if (( BASH_VERSINFO[0] < REQUIRED_BASH_MAJOR || ( BASH_VERSINFO[0] == REQUIRED_BASH_MAJOR && BASH_VERSINFO[1] < REQUIRED_BASH_MINOR ) )); then
    echo "Current Bash version is older than the required version. Installing Bash ${INSTALL_BASH_VERSION}..."
    ./perfmetrics/scripts/install_bash.sh "$INSTALL_BASH_VERSION"
    BASH_EXECUTABLE="/usr/local/bin/bash"
else
    echo "Current Bash version (${BASH_VERSINFO[0]}.${BASH_VERSINFO[1]}) meets or exceeds the required version (${REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT}). Skipping Bash installation."
fi

if $IS_ZONAL; then
  echo "Running zonal e2e tests on installed package...."
  "${BASH_EXECUTABLE}" ./tools/integration_tests/improved_run_e2e_tests.sh --bucket-location="$BUCKET_LOCATION" --test-installed-package --zonal
  exit 0
fi
echo "Running regional e2e tests on installed package...."
"${BASH_EXECUTABLE}" ./tools/integration_tests/improved_run_e2e_tests.sh --bucket-location="$BUCKET_LOCATION" --test-installed-package
