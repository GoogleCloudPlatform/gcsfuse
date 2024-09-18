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

cd "../"
./upgrade_gcloud.sh
cd "hns_rename_folders_metrics"

UPLOAD_FLAGS=$1
gsutil cp gs://periodic-perf-tests/creds.json ../gsheet/

echo "Upgrading gcloud version"
../upgrade_gcloud.sh

#echo "Running renaming benchmark on flat bucket"
#python3 renaming_benchmark.py config-flat.json flat "$UPLOAD_FLAGS"

echo "Running renaming benchmark on HNS bucket"
LOG_FILE_HNS=$2
python3 renaming_benchmark.py config-hns.json hns $UPLOAD_FLAGS --log_file $LOG_FILE_HNS
