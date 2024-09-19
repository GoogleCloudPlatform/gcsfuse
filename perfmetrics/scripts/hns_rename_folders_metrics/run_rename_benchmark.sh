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

set -e
echo Installing pip and fuse..
sudo apt-get install fuse -y
sudo apt-get install pip -y
echo Installing requirements..
pip install --require-hashes -r requirements.txt --user
echo Running script..

echo "Installing the Cloud Monitoring agent on VM ...."
curl -sSO https://dl.google.com/cloudagents/add-monitoring-agent-repo.sh
sudo bash add-monitoring-agent-repo.sh --also-install

echo "Installing Ops Agent on Vm"
curl -sSO https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh
sudo bash add-google-cloud-ops-agent-repo.sh --also-install

../upgrade_gcloud.sh

GCSFUSE_HNS_FLAGS=$1
GCSFUSE_FLAT_FLAGS=$2
UPLOAD_FLAGS=$3
gsutil cp gs://periodic-perf-tests/creds.json ../gsheet/

echo "Upgrading gcloud version"
../upgrade_gcloud.sh

# Uncomment the following 2 lines to run the benchmark on flat bucket
#echo "Running renaming benchmark on flat bucket"
#python3 renaming_benchmark.py config-flat.json flat "$GCSFUSE_FLAT_FLAGS" "$UPLOAD_FLAGS"

echo "Running renaming benchmark on HNS bucket"
python3 renaming_benchmark.py config-hns.json hns "$GCSFUSE_HNS_FLAGS" $UPLOAD_FLAGS
