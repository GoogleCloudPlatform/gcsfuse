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
echo "Installing fio"
# install libaio as fio has a dependency on libaio
sudo apt-get install libaio-dev
# We are building fio from source because of issue: https://github.com/axboe/fio/issues/1640.
# The fix is not currently released in a package as of 20th Oct, 2023.
# TODO: install fio via package when release > 3.35 is available.
sudo rm -rf "${KOKORO_ARTIFACTS_DIR}/github/fio"
git clone https://github.com/axboe/fio.git "${KOKORO_ARTIFACTS_DIR}/github/fio"
cd  "${KOKORO_ARTIFACTS_DIR}/github/fio" && \
git checkout c5d8ce3fc736210ded83b126c71e3225c7ffd7c9 && \
./configure && make && sudo make install

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"

echo "Mounting gcs bucket"
mkdir -p gcs
GCSFUSE_FLAGS=$1
UPLOAD_FLAGS=$2
BUCKET_NAME=periodic-perf-tests
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
gcsfuse $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT

echo Print the time when FIO tests start
date
echo Running fio test..
echo "Overall fio start epoch time:" `date +%s`
fio job_files/seq_rand_read_write.fio --lat_percentiles 1 --output-format=json --output='fio-output.json'
echo "Overall fio end epoch time:" `date +%s`
sudo umount $MOUNT_POINT

echo Installing requirements..
pip install --require-hashes -r requirements.txt --user
gsutil cp gs://periodic-perf-tests/creds.json gsheet
echo Fetching results..
python3 fetch_and_upload_metrics.py fio-output.json $UPLOAD_FLAGS
