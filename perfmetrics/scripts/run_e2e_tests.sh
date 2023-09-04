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

readonly INTEGRATION_TEST_EXECUTION_TIME=24m
# true or false to run e2e tests on installedPackage
run_e2e_tests_on_package=$1

# e.g. architecture=arm64 or amd64
architecture=$(dpkg --print-architecture)
echo "Installing git..."
sudo apt-get install git
echo "Installing go-lang 1.21.0..."
wget -O go_tar.tar.gz https://go.dev/dl/go1.21.0.linux-${architecture}.tar.gz -q
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin

# Create bucket for integration tests.
# The prefix for the random string
bucketPrefix="gcsfuse-integration-test-"
# The length of the random string
length=5
# Generate the random string
random_string=$(tr -dc 'a-z0-9' < /dev/urandom | head -c $length)
BUCKET_NAME=$bucketPrefix$random_string
echo 'bucket name = '$BUCKET_NAME
gcloud alpha storage buckets create gs://$BUCKET_NAME --project=gcs-fuse-test-ml --location=us-west1 --uniform-bucket-level-access

# Executing integration tests
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/... -p 1 --integrationTest -v --testbucket=$BUCKET_NAME --testInstalledPackage=$run_e2e_tests_on_package -timeout $INTEGRATION_TEST_EXECUTION_TIME

# Delete bucket after testing.
gcloud alpha storage rm --recursive gs://$BUCKET_NAME/
