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

# This will stop execution when any command will have non-zero status.

set -e

readonly RUN_E2E_TESTS_ON_INSTALLED_PACKAGE=true
readonly SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE=true
readonly RUN_TEST_ON_TPC_ENDPOINT=true
# TPC project id
readonly PROJECT_ID="tpczero-system:gcsfuse-test-project"
readonly BUCKET_LOCATION="u-us-prp1"

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
echo "Building and installing gcsfuse..."
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
./perfmetrics/scripts/build_and_install_gcsfuse.sh $commitId

## To execute tests for a specific commitId, ensure you've checked out that commitId first.
git checkout $commitId

sudo gcloud storage cp gs://gcsfuse-tpc-tests/creds.json /tmp/sa.key.json

architecture=$(dpkg --print-architecture)
# e.g. architecture=arm64 or amd64
echo "Installing git"
sudo apt-get install git
echo "Installing go-lang 1.20.5"
wget -O go_tar.tar.gz https://go.dev/dl/go1.20.5.linux-${architecture}.tar.gz -q
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin
echo "Installing docker "
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=${architecture} signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin -y

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
# integration tests using code upto that commit.
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
git stash
git checkout $commitId

echo "Building and installing gcsfuse"
# Build the gcsfuse package using the same commands used during release.
GCSFUSE_VERSION=0.0.0
sudo docker buildx build --load ./tools/package_gcsfuse_docker/ -t gcsfuse-release:$commitId --build-arg ARCHITECTURE=${architecture} --build-arg GCSFUSE_VERSION=$GCSFUSE_VERSION --build-arg BRANCH_NAME=$commitId --platform=linux/${architecture}
sudo docker run -v $HOME/release:/release gcsfuse-release:$commitId cp -r /packages /release/
sudo dpkg -i $HOME/release/packages/gcsfuse_${GCSFUSE_VERSION}_${architecture}.deb

echo "Executing integration tests"
GODEBUG=asyncpreemptoff=1 CGO_ENABLED=0 go test ./tools/integration_tests/... -p 1 --testInstalledPackage --integrationTest -v --testbucket=gcsfuse-integration-test -timeout 24m
