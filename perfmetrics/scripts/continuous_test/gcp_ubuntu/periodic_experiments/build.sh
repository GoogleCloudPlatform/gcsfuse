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
sudo apt-get update

echo "Installing git"
sudo apt-get install git

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"

echo "Install command-line JSON processing tool"
sudo apt-get install jq -y

# Get the current date and time
current_date=$(date +"%Y-%m-%d %H:%M:%S")

export EXPERIMENT_NUMBER=4
# Get the required enabled configuration
config=$(jq --arg EXPERIMENT_NUMBER "$EXPERIMENT_NUMBER" --arg current_date "$current_date" '.experiment_configuration[] | select(.end_date >= $current_date)' perfmetrics/scripts/continuous_test/gcp_ubuntu/periodic_experiments/experiments_configuration.json | jq -s ".[$EXPERIMENT_NUMBER-1]")

# Check if any enabled configurations were found
if [ "$config" = "null" ]; then
  echo "No enabled configuration found for value: $EXPERIMENT_NUMBER"
  exit 0
fi

# Access specific properties of the configuration
CONFIG_NAME=$(echo "$config" | jq -r '.config_name')
GCSFUSE_FLAGS=$(echo "$config" | jq -r '.gcsfuse_flags')
BRANCH=$(echo "$config" | jq -r '.branch')
END_DATE=$(echo "$config" | jq -r '.end_date')
# Get the value of the config_file_flag key
CONFIG_FILE_FLAG_JSON=$(jq -r '.["config_file_flag"]' <<< $config )

# Create config.yml file from json.
CONFIG_FILE_YML="${KOKORO_ARTIFACTS_DIR}/config_flags.yml"
CONFIG_FILE_JSON="${KOKORO_ARTIFACTS_DIR}/config_flags.json"
echo "$CONFIG_FILE_FLAG_JSON" >> $CONFIG_FILE_JSON
cat $CONFIG_FILE_JSON
if [ -n "$CONFIG_FILE_JSON" ];
then
  jq -c -M . $CONFIG_FILE_JSON > $CONFIG_FILE_YML
  GCSFUSE_FLAGS="$GCSFUSE_FLAGS --config-file $CONFIG_FILE_YML "
fi
CONFIG_FILE_STRING=$(cat $CONFIG_FILE_JSON | jq -c .)

echo "Building and installing gcsfuse"
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
chmod +x perfmetrics/scripts/build_and_install_gcsfuse.sh
./perfmetrics/scripts/build_and_install_gcsfuse.sh $BRANCH

cd "./perfmetrics/scripts/"
export PYTHONPATH="./"

echo Installing requirements..
pip install --require-hashes -r bigquery/requirements.txt --user

CONFIG_ID=$(eval "python3 -m bigquery.get_experiments_config --gcsfuse_flags '$GCSFUSE_FLAGS' --config_file_flag '$CONFIG_FILE_STRING' --branch '$BRANCH' --end_date '$END_DATE' --config_name '$CONFIG_NAME'")
START_TIME_BUILD=$(date +%s)

# Upload data to the gsheet only when it runs through kokoro.
UPLOAD_FLAGS=""
if [ "${KOKORO_JOB_TYPE}" == "RELEASE" ] || [ "${KOKORO_JOB_TYPE}" == "CONTINUOUS_INTEGRATION" ] || [ "${KOKORO_JOB_TYPE}" == "PRESUBMIT_GITHUB" ] || [ "${KOKORO_JOB_TYPE}" == "SUB_JOB" ];
then
  UPLOAD_FLAGS="--upload_gs --upload_bq --config_id $CONFIG_ID --start_time_build $START_TIME_BUILD"
fi

# Executing perf tests
LOG_FILE_FIO_TESTS="${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs${EXPERIMENT_NUMBER}.txt"

GCSFUSE_FIO_FLAGS="$GCSFUSE_FLAGS --log-file $LOG_FILE_FIO_TESTS --log-format \"text\" --stackdriver-export-interval=30s"
chmod +x run_load_test_and_fetch_metrics.sh
./run_load_test_and_fetch_metrics.sh "$GCSFUSE_FIO_FLAGS" "$UPLOAD_FLAGS"

# ls_metrics test. This test does gcsfuse mount with the passed flags first and then does the testing.
LOG_FILE_LIST_TESTS="${KOKORO_ARTIFACTS_DIR}/gcsfuse-list-logs${EXPERIMENT_NUMBER}.txt"
GCSFUSE_LIST_FLAGS="$GCSFUSE_FLAGS --log-file $LOG_FILE_LIST_TESTS --log-format \"text\" --stackdriver-export-interval=30s"
cd "./ls_metrics"
chmod +x run_ls_benchmark.sh
./run_ls_benchmark.sh "$GCSFUSE_LIST_FLAGS" "$UPLOAD_FLAGS"
rm $CONFIG_FILE_JSON $CONFIG_FILE_YML
