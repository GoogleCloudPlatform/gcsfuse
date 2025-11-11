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

# Running test only for when PR contains execute-perf-test,
# execute-integration-tests or execute-checkpoint-test label.
readonly EXECUTE_PERF_TEST_LABEL="execute-perf-test"
readonly EXECUTE_INTEGRATION_TEST_LABEL="execute-integration-tests"
readonly EXECUTE_INTEGRATION_TEST_LABEL_ON_ZB="execute-integration-tests-on-zb"
readonly EXECUTE_PACKAGE_BUILD_TEST_LABEL="execute-package-build-tests"
readonly EXECUTE_CHECKPOINT_TEST_LABEL="execute-checkpoint-test"
readonly BUCKET_LOCATION=us-west4
readonly GO_VERSION="1.24.10"
readonly REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT="5.1"
readonly INSTALL_BASH_VERSION="5.3" # Using 5.3 for installation as bash 5.1 has an installation bug.

curl https://api.github.com/repos/GoogleCloudPlatform/gcsfuse/pulls/$KOKORO_GITHUB_PULL_REQUEST_NUMBER >> pr.json
perfTest=$(grep "$EXECUTE_PERF_TEST_LABEL" pr.json)
integrationTests=$(grep "\"$EXECUTE_INTEGRATION_TEST_LABEL\"" pr.json)
integrationTestsOnZB=$(grep "\"$EXECUTE_INTEGRATION_TEST_LABEL_ON_ZB\"" pr.json)
packageBuildTests=$(grep "$EXECUTE_PACKAGE_BUILD_TEST_LABEL" pr.json)
checkpointTests=$(grep "$EXECUTE_CHECKPOINT_TEST_LABEL" pr.json)
rm pr.json
perfTestStr="$perfTest"
integrationTestsStr="$integrationTests"
integrationTestsOnZBStr="$integrationTestsOnZB"
packageBuildTestsStr="$packageBuildTests"
checkpointTestStr="$checkpointTests"


set -e
sudo apt-get update
echo Installing git
sudo apt-get install git
cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"

# Fetch PR branch
echo '[remote "origin"]
         fetch = +refs/pull/*/head:refs/remotes/origin/pr/*' >> .git/config
git fetch origin -q

function execute_perf_test() {
  mkdir -p gcs
  GCSFUSE_FLAGS="--implicit-dirs --prometheus-port=48341"
  BUCKET_NAME=presubmit-perf-tests
  MOUNT_POINT=gcs
  # The VM will itself exit if the gcsfuse mount fails.
  go run . $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
  # Running FIO test
  time ./perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh
  sudo umount gcs
}

function install_requirements() {
  # Installing requirements
  cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
}

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


./perfmetrics/scripts/continuous_test/gke/orbax_benchmark/run_benchmark.py --project_id gcs-fuse-test-ml --zone europe-west4-a --bucket llama_europe_west4  --no_cleanup --iterations=1