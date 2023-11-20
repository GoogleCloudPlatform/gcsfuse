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

echo "Installing pip"
sudo apt-get install pip -y
"${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts/fio/install_fio.sh" "${KOKORO_ARTIFACTS_DIR}/github"

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"

echo "Mounting gcs bucket"
mkdir -p gcs
GCSFUSE_FLAGS=$1
UPLOAD_FLAGS=$2
BUCKET_NAME=$3
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
gcsfuse $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT

echo Print the time when FIO tests start
date
echo Running fio test..
echo "Overall fio start epoch time:" `date +%s`
fio job_files/seq_rand_read_write.fio --lat_percentiles 1 --output-format=json --output="fio-output${EXPERIMENT_NUMBER}.json"
echo "Overall fio end epoch time:" `date +%s`
sudo umount $MOUNT_POINT

echo Installing requirements..
pip install --require-hashes -r requirements.txt --user
gsutil cp gs://periodic-perf-tests/creds.json gsheet
echo Fetching results..
python3 fetch_and_upload_metrics.py "fio-output${EXPERIMENT_NUMBER}.json" $UPLOAD_FLAGS
