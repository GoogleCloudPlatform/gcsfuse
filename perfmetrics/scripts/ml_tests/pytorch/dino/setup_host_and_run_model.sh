#!/bin/bash

# This will stop execution when any command will have non-zero status.
set -e

cd "$HOME/github/gcsfuse/perfmetrics/scripts"

echo "Setting up the machine with Docker and Nvidia Driver"
chmod +x ml_tests/setup_host.sh
source ml_tests/setup_host.sh

cd "$HOME/github/gcsfuse"
echo "Building docker image containing all pytorch libraries..."
sudo docker build . -f perfmetrics/scripts/ml_tests/pytorch/dino/Dockerfile --tag pytorch-gcsfuse

mkdir -p container_artifacts

echo "Running the docker image build in the previous step..."
sudo docker run --runtime=nvidia --name=pytorch_automation_container --privileged -d -v $HOME/github/gcsfuse/container_artifacts:/pytorch_dino/run_artifacts:rw,rshared \
--shm-size=128g pytorch-gcsfuse:latest

# Setup the log_rotation.
chmod +x perfmetrics/scripts/ml_tests/setup_log_rotation.sh
source perfmetrics/scripts/ml_tests/setup_log_rotation.sh $HOME/github/gcsfuse/container_artifacts/gcsfuse.log

# Wait for the script completion as well as logs output.
sudo docker logs -f pytorch_automation_container
