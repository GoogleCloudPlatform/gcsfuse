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

echo "Mounting gcs bucket"
mkdir -p gcs
GCSFUSE_FLAGS=$1
BUCKET_NAME=periodic-perf-tests
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
gcsfuse $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT

echo Print the time when FIO tests start
date
echo Running fio test..
fio job_files/seq_rand_read_write.fio --lat_percentiles 1 --output-format=json --output='fio-output.json'

echo Installing requirements..
pip install --require-hashes -r requirements.txt --user
gsutil cp gs://periodic-perf-tests/creds.json gsheet
echo Fetching results..
# Upload data to the gsheet only when it runs through kokoro.
if [ "${KOKORO_JOB_TYPE}" != "RELEASE" ] && [ "${KOKORO_JOB_TYPE}" != "CONTINUOUS_INTEGRATION" ] && [ "${KOKORO_JOB_TYPE}" != "PRESUBMIT_GITHUB" ];
then
  python3 fetch_metrics.py fio-output.json
else
  python3 fetch_metrics.py fio-output.json --upload
fi

sudo umount $MOUNT_POINT
