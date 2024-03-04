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

# Running test only for when PR contains execute-perf-test or execute-integration-tests label
readonly EXECUTE_PERF_TEST_LABEL="execute-perf-test"
readonly EXECUTE_INTEGRATION_TEST_LABEL="execute-integration-tests"
readonly RUN_E2E_TESTS_ON_INSTALLED_PACKAGE=false

curl https://api.github.com/repos/GoogleCloudPlatform/gcsfuse/pulls/$KOKORO_GITHUB_PULL_REQUEST_NUMBER >> pr.json
perfTest=$(grep "$EXECUTE_PERF_TEST_LABEL" pr.json)
integrationTests=$(grep "$EXECUTE_INTEGRATION_TEST_LABEL" pr.json)
rm pr.json
perfTestStr="$perfTest"
integrationTestsStr="$integrationTests"
if [[ "$perfTestStr" != *"$EXECUTE_PERF_TEST_LABEL"*  &&  "$integrationTestsStr" != *"$EXECUTE_INTEGRATION_TEST_LABEL"* ]]
then
  echo "No need to execute tests"
  exit 0
fi

set -e
sudo apt-get update
echo Installing git
sudo apt-get install git
echo Installing go-lang  1.22.0
wget -O go_tar.tar.gz https://go.dev/dl/go1.22.0.linux-amd64.tar.gz -q
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin
export CGO_ENABLED=0
cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
# Fetch PR branch
echo '[remote "origin"]
         fetch = +refs/pull/*/head:refs/remotes/origin/pr/*' >> .git/config
git fetch origin -q

function execute_perf_test() {
  mkdir -p gcs
  GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100"
  BUCKET_NAME=presubmit-perf-tests
  MOUNT_POINT=gcs
  # The VM will itself exit if the gcsfuse mount fails.
  go run . $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
  # Running FIO test
  ./perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh
  sudo umount gcs
}

function install_requirements() {
  # Installing requirements
  echo installing requirements
  echo Installing python3-pip
  sudo apt-get -y install python3-pip
  echo Installing Bigquery module requirements...
  pip install --require-hashes -r ./perfmetrics/scripts/bigquery/requirements.txt --user
  echo Installing libraries to run python script
  pip install google-cloud
  pip install google-cloud-vision
  pip install google-api-python-client
  pip install prettytable
  "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts/fio/install_fio.sh" "${KOKORO_ARTIFACTS_DIR}/github"
  cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
}

# execute perf tests.
if [[ "$perfTestStr" == *"$EXECUTE_PERF_TEST_LABEL"* ]];
then
 # Executing perf tests for master branch
 install_requirements
 git checkout master
 # Store results
 touch result.txt
 echo Mounting gcs bucket for master branch and execute tests
 execute_perf_test


 # Executing perf tests for PR branch
 echo checkout PR branch
 git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER
 echo Mounting gcs bucket from pr branch and execute tests
 execute_perf_test

 # Show results
 echo showing results...
 python3 ./perfmetrics/scripts/presubmit/print_results.py
fi

# Execute integration tests.
if [[ "$integrationTestsStr" == *"$EXECUTE_INTEGRATION_TEST_LABEL"* ]];
then
  echo checkout PR branch
  git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER

  echo "Running e2e tests...."
  # $1 argument is refering to value of testInstalledPackage.
  ./perfmetrics/scripts/run_e2e_tests.sh $RUN_E2E_TESTS_ON_INSTALLED_PACKAGE
fi
