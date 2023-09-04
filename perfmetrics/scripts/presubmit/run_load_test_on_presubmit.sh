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

set -e
# Installing requirements
echo Installing python3-pip
sudo apt-get -y install python3-pip
echo Installing libraries to run python script
pip install google-cloud
pip install google-cloud-vision
pip install google-api-python-client
pip install prettytable
echo Installing fio
sudo apt-get install fio -y

echo Running fio test..
fio ./perfmetrics/scripts/job_files/presubmit_perf_test.fio --lat_percentiles 1 --output-format=json --output='output.json'
echo fetching results..
python3 ./perfmetrics/scripts/presubmit/fetch_results.py output.json
