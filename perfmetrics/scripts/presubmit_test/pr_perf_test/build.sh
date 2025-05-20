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
readonly INSTALL_DIR="/bin"
readonly GCSFUSE_BINARY_NAME="gcsfuse"
readonly RUN_E2E_TESTS_ON_INSTALLED_PACKAGE=true
readonly SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE=false
readonly BUCKET_LOCATION=us-west4
readonly RUN_TEST_ON_TPC_ENDPOINT=false

# This flag, if set true, will indicate to underlying script to customize for a presubmit run.
readonly RUN_TESTS_WITH_PRESUBMIT_FLAG=true

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
if [[ "$perfTestStr" != *"$EXECUTE_PERF_TEST_LABEL"*  && "$integrationTestsStr" != *"$EXECUTE_INTEGRATION_TEST_LABEL"*  && "$integrationTestsOnZBStr" != *"$EXECUTE_INTEGRATION_TEST_LABEL_ON_ZB"*  && "$packageBuildTestsStr" != *"$EXECUTE_PACKAGE_BUILD_TEST_LABEL"* && "$checkpointTestStr" != *"$EXECUTE_CHECKPOINT_TEST_LABEL"* ]]
then
  echo "No need to execute tests"
  exit 0
fi

set -e
sudo apt-get update
BASH_VER="5.2.37"
INSTALL_PREFIX="/usr/local"
wget -q "https://ftp.gnu.org/gnu/bash/bash-${BASH_VER}.tar.gz"
tar -xzf "bash-${BASH_VER}.tar.gz"
cd "bash-${BASH_VER}"
./configure --prefix="${INSTALL_PREFIX}" --enable-readline > /dev/null 2>&1
make -s -j"$(nproc 2>/dev/null || echo 1)" > /dev/null 2>&1
sudo make install > /dev/null 2>&1
echo ""
echo "Bash ${BASH_VER} installed to ${INSTALL_PREFIX}/bin/bash"
"${INSTALL_PREFIX}/bin/bash" --version
echo ""
echo Installing git
sudo apt-get install git
echo Installing go-lang  1.24.0
wget -O go_tar.tar.gz https://go.dev/dl/go1.24.0.linux-amd64.tar.gz -q
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin
export CGO_ENABLED=0
cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
# Fetch PR branch
echo '[remote "origin"]
         fetch = +refs/pull/*/head:refs/remotes/origin/pr/*' >> .git/config
git fetch origin -q

install_gcsfuse() {
  if [[ $# -ne 1 ]]; then
    echo "Error: install_gcsfuse() called with incorrect number of arguments."
    echo "Usage: install_gcsfuse <PR>"
    exit 0
  fi
  local pr="$1"
  echo "Installing GCSFuse at PR: ${pr}"
  git checkout "$pr"
  echo "Building gcsfuse..."
  local src_dir=$(pwd)
  local dst_dir=$(mktemp -d -t gcsfuse_dst_dir_XXXXXX)
  go run ./tools/build_gcsfuse/main.go "$src_dir" "$dst_dir" "${pr}" || { echo "Go build failed"; exit 1; }
  export PATH="$dst_dir/bin:$dst_dir/sbin:$PATH"
  sudo cp "/$dst_dir/bin/gcsfuse" /bin/
  sudo cp "/$dst_dir/sbin/mount.gcsfuse" /sbin/
  sudo cp "/$dst_dir/sbin/mount.fuse.gcsfuse" /sbin/
  sudo chmod 755 /bin/gcsfuse
  sudo chmod 755 /sbin/mount.gcsfuse
  sudo chmod 755 /sbin/mount.fuse.gcsfuse
  sudo chown root:root /bin/gcsfuse
  sudo chown root:root /sbin/mount.gcsfuse
  sudo chown root:root /sbin/mount.fuse.gcsfuse
  whereis gcsfuse
  which gcsfuse
  whereis mount.gcsfuse
  which mount.gcsfuse
  whereis mount.fuse.gcsfuse
  which mount.fuse.gcsfuse
  echo "gcsfuse installed Successfully"
}

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

# Execute integration tests on zonal bucket(s).
if test -n "${integrationTestsOnZBStr}" ;
then
  install_gcsfuse "pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER"
  echo checkout PR branch
  git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER

  echo "Running e2e tests on zonal bucket(s) ..."
  # $1 argument is refering to value of testInstalledPackage.
  "${INSTALL_PREFIX}/bin/bash" ./tools/integration_tests/run_e2e_tests.sh $RUN_E2E_TESTS_ON_INSTALLED_PACKAGE $SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE $BUCKET_LOCATION $RUN_TEST_ON_TPC_ENDPOINT $RUN_TESTS_WITH_PRESUBMIT_FLAG true
fi

# Execute integration tests on non-zonal bucket(s).
if test -n "${integrationTestsStr}" ;
then
  install_gcsfuse "pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER"
  echo checkout PR branch
  git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER

  echo "Running e2e tests on non-zonal bucket(s) ..."
  # $1 argument is refering to value of testInstalledPackage.
  "${INSTALL_PREFIX}/bin/bash" ./tools/integration_tests/run_e2e_tests.sh $RUN_E2E_TESTS_ON_INSTALLED_PACKAGE $SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE $BUCKET_LOCATION $RUN_TEST_ON_TPC_ENDPOINT $RUN_TESTS_WITH_PRESUBMIT_FLAG false
fi

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
