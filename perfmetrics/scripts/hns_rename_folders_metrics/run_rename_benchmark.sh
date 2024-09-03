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
# Refer for env var: https://stackoverflow.com/questions/72441758/typeerror-descriptors-cannot-not-be-created-directly
export PROTOCOL_BUFFERS_PYTHON_IMPLEMENTATION=python
echo Installing pip and fuse..
sudo apt-get install fuse -y
sudo apt-get install pip -y
echo Installing requirements..
pip install --require-hashes -r requirements.txt --user
echo Running script..

echo "Installing the Cloud Monitoring agent on VMs ...."
curl -sSO https://dl.google.com/cloudagents/add-monitoring-agent-repo.sh
sudo bash add-monitoring-agent-repo.sh --also-install

CONFIG_FILE_HNS=$1
CONFIG_FILE_FLAt=$1

gsutil cp gs://periodic-perf-tests/creds.json ./gsheet
echo "Running renaming benchmark on flat bucket"
python3 renaming_benchmark.py $CONFIG_FILE_FLAT flat --upload_gs

echo "Running renaming benchmark on HNS bucket"
python3 renaming_benchmark.py $CONFIG_FILE_HNS hns --upload_gs