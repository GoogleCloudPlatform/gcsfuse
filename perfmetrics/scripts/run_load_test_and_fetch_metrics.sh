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
echo "Installing fio"
sudo apt-get install fio -y
echo "Installing pip"
sudo apt-get install pip -y
echo Print the time when FIO tests start
date
echo Running fio test..

# Run individual jobs for each .fio job file in job_files/seq_rand_read_write/*_thread.fio.
# Create a separate fio-output json file for each, record their names 
# in an array, and calculate metrics and upload each individually.
fio_output_jsons=()
echo "Overall fio start epoch time:" `date +%s`
for job_group_path in job_files/seq_rand_read_write/*_thread.fio; do
  # Wait for 2 minutes to keep a gap between consecutive job-groups.
  sleep 120
  job_group_name=$(basename "$job_group_path")
  echo "Start epoch time for "${job_group_name}" :" `date +%s`
  fio_output_json='fio-output-'${job_group_name}'.json'
  fio ${job_group_path} --lat_percentiles 1 --output-format=json --output=${fio_output_json}
  fio_output_jsons+=${fio_output_json}
  echo "End epoch time for "${job_group_name}" :" `date +%s`
done
echo "Overall fio end epoch time:" `date +%s`

echo Installing requirements..
pip install --require-hashes -r requirements.txt --user
gsutil cp gs://periodic-perf-tests/creds.json gsheet
echo Fetching results..

ARGS=
# Upload data to the gsheet only when it runs through kokoro.
if [ "${KOKORO_JOB_TYPE}" == "RELEASE" ] || [ "${KOKORO_JOB_TYPE}" == "CONTINUOUS_INTEGRATION" ] || [ "${KOKORO_JOB_TYPE}" == "PRESUBMIT_GITHUB" ]; then
  ARGS="--upload"
fi

for fio_output_json in ${fio_output_jsons[@]}; do
  python3 fetch_metrics.py ${fio_output_json} $ARGS
done
