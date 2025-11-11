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

#curl https://api.github.com/repos/GoogleCloudPlatform/gcsfuse/pulls/$KOKORO_GITHUB_PULL_REQUEST_NUMBER >> pr.json
#perfTest=$(grep "$EXECUTE_PERF_TEST_LABEL" pr.json)
#integrationTests=$(grep "\"$EXECUTE_INTEGRATION_TEST_LABEL\"" pr.json)
#integrationTestsOnZB=$(grep "\"$EXECUTE_INTEGRATION_TEST_LABEL_ON_ZB\"" pr.json)
#packageBuildTests=$(grep "$EXECUTE_PACKAGE_BUILD_TEST_LABEL" pr.json)
#checkpointTests=$(grep "$EXECUTE_CHECKPOINT_TEST_LABEL" pr.json)
#rm pr.json
#perfTestStr="$perfTest"
#integrationTestsStr="$integrationTests"
#integrationTestsOnZBStr="$integrationTestsOnZB"
#packageBuildTestsStr="$packageBuildTests"
#checkpointTestStr="$checkpointTests"
#if [[ "$perfTestStr" != *"$EXECUTE_PERF_TEST_LABEL"*  && "$integrationTestsStr" != *"$EXECUTE_INTEGRATION_TEST_LABEL"*  && "$integrationTestsOnZBStr" != *"$EXECUTE_INTEGRATION_TEST_LABEL_ON_ZB"*  && "$packageBuildTestsStr" != *"$EXECUTE_PACKAGE_BUILD_TEST_LABEL"* && "$checkpointTestStr" != *"$EXECUTE_CHECKPOINT_TEST_LABEL"* ]]
#then
#  echo "No need to execute tests"
#  exit 0
#fi

set -e
sudo apt-get update
echo Installing git
sudo apt-get install git
cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
# Install required go version.
./perfmetrics/scripts/install_go.sh "$GO_VERSION"
export CGO_ENABLED=0
export PATH="/usr/local/go/bin:$PATH"

# Install latest gcloud version for compatability with HNS bucket.
./perfmetrics/scripts/install_latest_gcloud.sh
export PATH="/usr/local/google-cloud-sdk/bin:$PATH"

#Running the rename benchmark script.
cd "./perfmetrics/scripts/"
cd "./hns_rename_folders_metrics"
./run_rename_benchmark.sh $UPLOAD_FLAGS

## Fetch PR branch
#echo '[remote "origin"]
#         fetch = +refs/pull/*/head:refs/remotes/origin/pr/*' >> .git/config
#git fetch origin -q
#
#function execute_perf_test() {
#  mkdir -p gcs
#  GCSFUSE_FLAGS="--implicit-dirs --prometheus-port=48341"
#  BUCKET_NAME=presubmit-perf-tests
#  MOUNT_POINT=gcs
#  # The VM will itself exit if the gcsfuse mount fails.
#  go run . $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
#  # Running FIO test
#  time ./perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh
#  sudo umount gcs
#}
#
#function install_requirements() {
#  # Installing requirements
#  echo installing requirements
#  echo Installing python3-pip
#  sudo apt-get -y install python3-pip
#  echo Installing Bigquery module requirements...
#  pip install --require-hashes -r ./perfmetrics/scripts/bigquery/requirements.txt --user
#  echo Installing libraries to run python script
#  pip install google-cloud
#  pip install google-cloud-vision
#  pip install google-api-python-client
#  pip install prettytable
#  "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts/fio/install_fio.sh" "${KOKORO_ARTIFACTS_DIR}/github"
#  cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
#}
#
## execute perf tests.
#if [[ "$perfTestStr" == *"$EXECUTE_PERF_TEST_LABEL"* ]];
#then
# # Executing perf tests for master branch
# install_requirements
# git checkout master
# # Store results
# touch result.txt
# echo Mounting gcs bucket for master branch and execute tests
# execute_perf_test
#
#
# # Executing perf tests for PR branch
# echo checkout PR branch
# git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER
# echo Mounting gcs bucket from pr branch and execute tests
# execute_perf_test
#
# # Show results
# echo showing results...
# python3 ./perfmetrics/scripts/presubmit/print_results.py
#fi
#
## Check and install required bash version for e2e script.
#BASH_EXECUTABLE="bash"
#REQUIRED_BASH_MAJOR=$(echo "$REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT" | cut -d'.' -f1)
#REQUIRED_BASH_MINOR=$(echo "$REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT" | cut -d'.' -f2)
#
#echo "Current Bash version: ${BASH_VERSINFO[0]}.${BASH_VERSINFO[1]}"
#echo "Required Bash version for e2e script: ${REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT}"
#
#if (( BASH_VERSINFO[0] < REQUIRED_BASH_MAJOR || ( BASH_VERSINFO[0] == REQUIRED_BASH_MAJOR && BASH_VERSINFO[1] < REQUIRED_BASH_MINOR ) )); then
#    echo "Current Bash version is older than the required version. Installing Bash ${INSTALL_BASH_VERSION}..."
#    ./perfmetrics/scripts/install_bash.sh "$INSTALL_BASH_VERSION"
#    BASH_EXECUTABLE="/usr/local/bin/bash"
#else
#    echo "Current Bash version (${BASH_VERSINFO[0]}.${BASH_VERSINFO[1]}) meets or exceeds the required version (${REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT}). Skipping Bash installation."
#fi
#
## Execute integration tests on zonal bucket(s).
#if test -n "${integrationTestsOnZBStr}" ;
#then
#  echo checkout PR branch
#  git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER
#
#  echo "Running e2e tests on zonal bucket(s) ..."
#  ${BASH_EXECUTABLE} ./tools/integration_tests/improved_run_e2e_tests.sh --bucket-location=$BUCKET_LOCATION --presubmit --zonal --track-resource-usage
#fi
#
## Execute integration tests on non-zonal bucket(s).
#if test -n "${integrationTestsStr}" ;
#then
#  echo checkout PR branch
#  git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER
#
#  echo "Running e2e tests on non-zonal bucket(s) ..."
#  ${BASH_EXECUTABLE} ./tools/integration_tests/improved_run_e2e_tests.sh --bucket-location=$BUCKET_LOCATION --presubmit --track-resource-usage
#fi
#
## Execute package build tests.
#if [[ "$packageBuildTestsStr" == *"$EXECUTE_PACKAGE_BUILD_TEST_LABEL"* ]];
#then
#  echo checkout PR branch
#  git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER
#
#  echo "Running package build tests...."
#  ./perfmetrics/scripts/build_and_install_gcsfuse.sh master
#fi
#
## Execute JAX checkpoints tests.
#if [[ "$checkpointTestStr" == *"$EXECUTE_CHECKPOINT_TEST_LABEL"* ]];
#then
#  echo checkout PR branch
#  git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER
#
#  echo "Running checkpoint tests...."
#  ./perfmetrics/scripts/ml_tests/checkpoint/Jax/run_checkpoints.sh
#fi
