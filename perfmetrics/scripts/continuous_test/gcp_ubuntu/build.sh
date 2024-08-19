#!/bin/bash
# Copyright 2023 Google LLC
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

# Mounting gcs bucket
cd "./perfmetrics/scripts/"

echo Installing Bigquery module requirements...
pip install --require-hashes -r bigquery/requirements.txt --user

# Upload data to the gsheet only when it runs through kokoro.
UPLOAD_FLAGS=""
if [ "${KOKORO_JOB_TYPE}" == "RELEASE" ] || [ "${KOKORO_JOB_TYPE}" == "CONTINUOUS_INTEGRATION" ] || [ "${KOKORO_JOB_TYPE}" == "PRESUBMIT_GITHUB" ] || [ "${KOKORO_JOB_TYPE}" == "SUB_JOB" ];
then
  UPLOAD_FLAGS="--upload_gs"
fi

run_load_test_and_fetch_metrics(){
  fio_flags=$1
  gcsfuse_flags="$COMMON_MOUNT_FLAGS $fio_flags"
  bucket_name=$2
  spreadsheet_id=$3

  # Executing perf tests
  ./run_load_test_and_fetch_metrics.sh "$gcsfuse_flags" "$UPLOAD_FLAGS" "$bucket_name" "$spreadsheet_id"
}

run_ls_benchmark(){
  ls_flags=$1
  gcsfuse_flags="$COMMON_MOUNT_FLAGS $ls_flags"
  spreadsheet_id="$2"
  config_file="$3"

  cd "./ls_metrics"
  ./run_ls_benchmark.sh "$gcsfuse_flags" "$UPLOAD_FLAGS" "$spreadsheet_id" "$config_file"
  cd "../"
}

COMMON_MOUNT_FLAGS="--debug_fuse --debug_gcs --log-format \"text\""

# Testing for flat bucket.
LOG_FILE_FIO_TESTS=${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-flat.txt
GCSFUSE_FIO_FLAGS="--implicit-dirs --stackdriver-export-interval=30s --log-file $LOG_FILE_FIO_TESTS"
GCSFUSE_LS_FLAGS="--implicit-dirs"
BUCKET_NAME="periodic-perf-tests"
SPREADSHEET_ID='1kvHv1OBCzr9GnFxRu9RTJC7jjQjc9M4rAiDnhyak2Sg'
LIST_CONFIG_FILE="config.json"
run_load_test_and_fetch_metrics "$GCSFUSE_FIO_FLAGS" "$BUCKET_NAME" "$SPREADSHEET_ID"
run_ls_benchmark "$GCSFUSE_LS_FLAGS" "$SPREADSHEET_ID" "$LIST_CONFIG_FILE"

# Testing for hns bucket.
echo "enable-hns: true
metadata-cache:
  ttl-secs: 0" > /tmp/config.yml
LOG_FILE_FIO_TESTS=${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-hns.txt
GCSFUSE_FIO_FLAGS="--config-file=/tmp/config.yml --stackdriver-export-interval=30s --log-file $LOG_FILE_FIO_TESTS"
GCSFUSE_LS_FLAGS="--config-file=/tmp/config.yml"
BUCKET_NAME="periodic-perf-tests-hns"
SPREADSHEET_ID='1wXRGYyAWvasU8U4KaP7NGPHEvgiOSgMd1sCLxsQUwf0'
LIST_CONFIG_FILE="config-hns.json"
run_load_test_and_fetch_metrics "$GCSFUSE_FIO_FLAGS" "$BUCKET_NAME" "$SPREADSHEET_ID"
run_ls_benchmark "$GCSFUSE_LS_FLAGS" "$SPREADSHEET_ID" "$LIST_CONFIG_FILE"
