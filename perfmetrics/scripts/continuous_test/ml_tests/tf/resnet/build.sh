#!/bin/bash

# This will stop execution when any command will have non-zero status.
set -e

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"

echo "Setting up the machine with Docker and Nvidia Driver..."
chmod +x ml_tests/setup_host.sh
source ml_tests/setup_host.sh

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/"
mkdir container_artifacts && mkdir container_artifacts/logs && mkdir container_artifacts/output

echo "Building tf DLC docker image containing all tensorflow libraries..."
sudo docker build . -f perfmetrics/scripts/ml_tests/tf/resnet/Dockerfile -t tf-dlc-gcsfuse --build-arg DLC_IMAGE_NAME=tf-gpu.2-10

echo "Running the docker image build in the previous step..."
sudo docker run --runtime=nvidia --name tf_model_container --privileged -d \
-v ${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/container_artifacts/logs:/home/logs:rw,rshared \
-v ${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/container_artifacts/output:/home/output:rw,rshared --shm-size=24g tf-dlc-gcsfuse:latest

echo "Setting up cron job to delete the gcsfuse_logs..."
echo "0 */1 * * * sudo sh ./perfmetrics/scripts/ml_tests/smart_log_deleter.sh container_artifacts/logs/" | crontab -

# Wait for the script completion as well as logs output.
sudo docker logs -f tf_model_container
