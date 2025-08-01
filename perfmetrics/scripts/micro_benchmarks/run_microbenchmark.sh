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

sudo apt-get install  git
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
cd gcsfuse
echo "Building and installing gcsfuse"
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
./perfmetrics/scripts/build_and_install_gcsfuse.sh $commitId

echo "Upgrading Python3 version"
./perfmetrics/scripts/upgrade_python3.sh

# Path to locally installed upgraded Python
PYTHON_BIN="$HOME/.local/python-3.11.9/bin/python3.11"

cd "./perfmetrics/scripts/micro_benchmarks"

echo "Installing dependencies using upgraded Python..."
"$PYTHON_BIN" -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# Temporarily allow script to continue after command failure
set +e

echo "Running Python scripts for hns bucket..."

FILE_SIZE_READ_GB=15
READ_LOG_FILE="${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-single-threaded-read-${FILE_SIZE_READ_GB}gb-test.txt"
GCSFUSE_READ_FLAGS="--log-file $READ_LOG_FILE"
python3 read_single_thread.py --bucket single-threaded-tests --gcsfuse-config "$GCSFUSE_READ_FLAGS" --total-files 10 --file-size-gb "$FILE_SIZE_READ_GB"
exit_read_code=$?

FILE_SIZE_WRITE_GB=15
WRITE_LOG_FILE="${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-single-threaded-write-${FILE_SIZE_WRITE_GB}gb-test.txt"
GCSFUSE_WRITE_FLAGS="--log-file $WRITE_LOG_FILE"
python3 write_single_thread.py --bucket single-threaded-tests --gcsfuse-config "$GCSFUSE_WRITE_FLAGS" --total-files 1 --file-size-gb "$FILE_SIZE_WRITE_GB"
exit_write_code=$?

deactivate

# Re-enable strict mode
set -e

# Final result
exit_code=0
if [[ $exit_read_code -ne 0 ]]; then
  echo "Read benchmark failed with exit code $exit_read_code"
  exit_code=$exit_read_code
fi

if [[ $exit_write_code -ne 0 ]]; then
  echo "Write benchmark failed with exit code $exit_write_code"
  exit_code=$exit_write_code
fi

if [[ $exit_code != 0 ]]; then
  echo "Benchmarks failed."
  exit $exit_code
fi

echo "Benchmarks completed successfully."
exit 0
