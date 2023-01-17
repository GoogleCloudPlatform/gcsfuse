#!/bin/bash

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts/continuous_test"

echo "Setting up the machine with Docker and Nvidia Driver"
chmod +x ml_tests/tf/setup_scripts/setup_host.sh
source ml_tests/tf/setup_scripts/setup_host.sh

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/"

mkdir container_artifacts && mkdir container_artifacts/logs && mkdir container_artifacts/output

# Building the dlc based dockerfile
sudo docker build . -f perfmetrics/scripts/continuous_test/ml_tests/tf/setup_scripts/Dockerfile -t dlc-gcsfuse --build-arg DLC_IMAGE_NAME=tf-gpu.2-10

# Running the container image
sudo docker run --runtime=nvidia --name tf_model_container --privileged -d \
-v container_artifacts/logs:/home/logs:rw,rshared \
-v container_artifacts/output:/home/output:rw,rshared --shm-size=24g dlc-gcsfuse:latest

chmod +x perfmetrics/scripts/continuous_test/ml_tests/tf/setup_scripts/resnet_run.sh
source perfmetrics/scripts/continuous_test/ml_tests/tf/setup_scripts/resnet_run.sh

# TODO cron setup once logging code is merged

sudo docker logs -f tf_model_container

# TODO: copy logs to a bucket
