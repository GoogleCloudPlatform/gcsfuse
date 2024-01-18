#!/bin/bash

# This will stop execution when any command will have non-zero status.
set -e

cd "$HOME/github/gcsfuse/perfmetrics/scripts"

echo "Setting up the machine with Docker and Nvidia Driver..."
DRIVER_VERSION="520.61.05"
source ml_tests/setup_host.sh $DRIVER_VERSION

cd "$HOME/github/gcsfuse/"
mkdir container_artifacts && mkdir container_artifacts/logs && mkdir container_artifacts/output

echo "Building tf DLC docker image containing all tensorflow libraries..."
sudo docker build . -f perfmetrics/scripts/ml_tests/tf/resnet/Dockerfile -t tf-dlc-gcsfuse --build-arg DLC_IMAGE_NAME=tf-gpu.2-13

echo "Running the docker image build in the previous step..."
sudo docker run --gpus all --name tf_model_container --privileged -d \
-v $HOME/github/gcsfuse/container_artifacts/logs:/home/logs:rw,rshared \
-v $HOME/github/gcsfuse/container_artifacts/output:/home/output:rw,rshared --shm-size=24g tf-dlc-gcsfuse:latest

# Wait for the script completion as well as logs output.
sudo docker logs -f tf_model_container
