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
readonly GO_VERSION="1.24.5"
readonly REQUIRED_BASH_VERSION_FOR_E2E_SCRIPT="5.3"

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

echo checkout PR branch
git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER

echo "Running e2e tests on non-zonal bucket(s) ..."
# $1 argument is refering to value of testInstalledPackage.
/usr/local/bin/bash ./tools/integration_tests/improved_run_e2e_tests.sh --bucket-location=$BUCKET_LOCATION --presubmit --track-resource-usage

# Execute package build tests.
if [[ "$packageBuildTestsStr" == *"$EXECUTE_PACKAGE_BUILD_TEST_LABEL"* ]];
then
  echo checkout PR branch
  git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER

  echo "Running package build tests...."
  ./perfmetrics/scripts/build_and_install_gcsfuse.sh master
fi

# Execute JAX checkpoints tests.
if [[ "$checkpointTestStr" == *"$EXECUTE_CHECKPOINT_TEST_LABEL"* ]];
then
  echo checkout PR branch
  git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER

  echo "Running checkpoint tests...."
  ./perfmetrics/scripts/ml_tests/checkpoint/Jax/run_checkpoints.sh
fi