#!/bin/bash
# Copyright 2025 Google LLC
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
sudo apt-get update

echo "Installing git"
sudo apt-get install git

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
echo "Building and installing gcsfuse"
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
./perfmetrics/scripts/build_and_install_gcsfuse.sh $commitId

cd "./perfmetrics/scripts/single_threaded_benchmark"

echo "Installing dependencies..."
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt

echo "Running Python scripts for hns bucket..."
FILE_SIZE_READ_GB=15
LOG_FILE=${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-single-threaded-read-${FILE_SIZE_READ_GB}gb-test.txt
GCSFUSE_READ_FLAGS="--log-file $LOG_FILE"
python3 read_single_thread.py  --bucket single-threaded-tests --gcsfuse-config "$GCSFUSE_READ_FLAGS" --total-files 10 --file-size-gb ${FILE_SIZE_READ_GB}

FILE_SIZE_WRITE_GB=15
LOG_FILE=${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-single-threaded-write-${FILE_SIZE_WRITE_GB}gb-test.txt
GCSFUSE_WRITE_FLAGS="--log-file $LOG_FILE"
python3 write_single_thread.py --bucket single-threaded-tests --gcsfuse-config "$GCSFUSE_WRITE_FLAGS" --total-files 1 --file-size-gb ${FILE_SIZE_WRITE_GB}

deactivate
