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
sudo apt-get update

# e.g. architecture=arm64 or amd64
architecture=$(dpkg --print-architecture)
echo "Installing git..."
sudo apt-get install git
echo "Installing go-lang 1.20.5..."
wget -O go_tar.tar.gz https://go.dev/dl/go1.20.5.linux-${architecture}.tar.gz -q
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin
echo "Installing docker..."
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
git checkout $commitId

echo "Building and installing gcsfuse..."
# Build the gcsfuse package using the same commands used during release.
GCSFUSE_VERSION=0.0.0
sudo docker buildx build --load ./tools/package_gcsfuse_docker/ -t gcsfuse:$commitId --build-arg ARCHITECTURE=${architecture} --build-arg GCSFUSE_VERSION=$GCSFUSE_VERSION --build-arg BRANCH_NAME=$commitId --platform=linux/${architecture}
sudo docker run -v $HOME/release:/release gcsfuse:$commitId cp -r /packages /release/
sudo dpkg -i $HOME/release/packages/gcsfuse_${GCSFUSE_VERSION}_${architecture}.deb

x=$((RANDOM % 10 + 1))
bucket_name=gcs-fuse-e2e-test-kokoro-${architecture}-${x}
echo "Creating buket for e2e tests..."
gcloud storage buckets create gs://${bucket_name} --project="gcs-fuse-test-ml" --location=us-west1 --uniform-bucket-level-access
echo "Executing integration tests..."
GODEBUG=asyncpreemptoff=1 CGO_ENABLED=0 go test ./tools/integration_tests/... -p 1 --testInstalledPackage --integrationTest -v --testbucket=${bucket_name} -timeout 24m
echo "Deleting bucket after testing..."
gcloud storage rm --recursive gs://${bucket_name}/
