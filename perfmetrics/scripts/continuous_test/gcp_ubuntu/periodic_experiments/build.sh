#!/bin/bash
set -e
sudo apt-get update

echo "Installing git"
sudo apt-get install git
echo "Installing pip"
sudo apt-get install pip -y
echo "Installing go-lang 1.20.4"
wget -O go_tar.tar.gz https://go.dev/dl/go1.20.4.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin
echo "Installing fio"
sudo apt-get install fio -y
echo "Installing docker "
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin -y
sudo apt-get install jq -y

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"

# Get the current date and time
current_date=$(date +"%Y-%m-%d %H:%M:%S")

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

echo "Building and installing gcsfuse"
# Build the gcsfuse package using the same commands used during release.
GCSFUSE_VERSION=0.0.0
sudo docker build ./tools/package_gcsfuse_docker/ -t gcsfuse:$BRANCH --build-arg GCSFUSE_VERSION=$GCSFUSE_VERSION --build-arg BRANCH_NAME=$BRANCH
sudo docker run -v $HOME/release:/release gcsfuse:$BRANCH cp -r /packages /release/
sudo dpkg -i $HOME/release/packages/gcsfuse_${GCSFUSE_VERSION}_amd64.deb

cd "./perfmetrics/scripts/"
export PYTHONPATH="./"

echo Installing requirements..
pip install --require-hashes -r bigquery/requirements.txt --user

CONFIG_ID=$(eval "python3 -m bigquery.get_experiments_config --gcsfuse_flags '$GCSFUSE_FLAGS' --branch '$BRANCH' --end_date '$END_DATE' --config_name '$CONFIG_NAME'")
START_TIME_BUILD=$(date +%s)

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
