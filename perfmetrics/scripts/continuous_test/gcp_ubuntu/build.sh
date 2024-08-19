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
  gcsfuse_fio_flags=$1
  upload_flags=$2
  bucket_name=$3
  spreadsheet_id=$4
  gsheet_credentials=$5
  test_type=$6
  gcsfuse_ls_flags=$7
  ls_config_file=$8

  # Executing perf tests
  ./run_load_test_and_fetch_metrics.sh "$gcsfuse_fio_flags" "$upload_flags" "$bucket_name" "$spreadsheet_id" "$gsheet_credentials"

  # ls_metrics test. This test does gcsfuse mount with the passed flags first and then does the testing.
  log_file_list_tests=${KOKORO_ARTIFACTS_DIR}/gcsfuse-list-logs-$test_type.txt
  gcsfuse_list_flags="$gcsfuse_ls_flags --log-file $log_file_list_tests"
  cd "./ls_metrics"
  ./run_ls_benchmark.sh "$gcsfuse_list_flags" "$upload_flags" "$spreadsheet_id" "$ls_config_file"
  cd "../"
}

GCSFUSE_FLAGS="--implicit-dirs  --debug_fuse --debug_gcs --log-format \"text\" "
LOG_FILE_FIO_TESTS=${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-flat.txt
GCSFUSE_FIO_FLAGS="$GCSFUSE_FLAGS --log-file $LOG_FILE_FIO_TESTS --stackdriver-export-interval=30s"
BUCKET_NAME="periodic-perf-tests"
SPREADSHEET_ID='1kvHv1OBCzr9GnFxRu9RTJC7jjQjc9M4rAiDnhyak2Sg'
GSHEET_CREDENTIALS="gs://periodic-perf-tests/creds.json"
LIST_CONFIG_FILE="config.json"
run_load_test_and_fetch_metrics "$GCSFUSE_FIO_FLAGS" "$UPLOAD_FLAGS" "$BUCKET_NAME" "$SPREADSHEET_ID" "$GSHEET_CREDENTIALS" "flat" "$GCSFUSE_FLAGS" "$LIST_CONFIG_FILE"

touch config.yml
echo "enable-hns: true
metadata-cache:
  ttl-secs: 0" > config.yml
COMMON_MOUNT_FLAGS="--debug_fuse --debug_gcs --log-format \"text\""
GCSFUSE_LS_FLAGS="--config-file=../config.yml $COMMON_MOUNT_FLAGS"
LOG_FILE_FIO_TESTS=${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs-hns.txt
GCSFUSE_FIO_FLAGS="--config-file=config.yml $COMMON_MOUNT_FLAGS --log-file $LOG_FILE_FIO_TESTS --stackdriver-export-interval=30s"
BUCKET_NAME="periodic-perf-tests-hns"
SPREADSHEET_ID='1wXRGYyAWvasU8U4KaP7NGPHEvgiOSgMd1sCLxsQUwf0'
GSHEET_CREDENTIALS="gs://periodic-perf-tests-hns/creds.json"
LIST_CONFIG_FILE="config-hns.json"
run_load_test_and_fetch_metrics "$GCSFUSE_FIO_FLAGS" "$UPLOAD_FLAGS" "$BUCKET_NAME" "$SPREADSHEET_ID" "$GSHEET_CREDENTIALS" "hns" "$GCSFUSE_LS_FLAGS" "$LIST_CONFIG_FILE"
rm config.yml
