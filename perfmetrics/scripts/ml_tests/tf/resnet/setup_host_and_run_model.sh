#!/bin/bash
# Copyright 2024 Google LLC
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

# This will stop execution when any command will have non-zero status.
set -e

BUCKET_TYPE=$1
cd "$HOME/github/gcsfuse/perfmetrics/scripts"

echo "Setting up the machine with Docker and Nvidia Driver..."
DRIVER_VERSION="550.127.05"
source ml_tests/setup_host.sh $DRIVER_VERSION

cd "$HOME/github/gcsfuse/"
mkdir container_artifacts && mkdir container_artifacts/logs && mkdir container_artifacts/output

echo "Building tf DLC docker image containing all tensorflow libraries..."
sudo docker build . -f perfmetrics/scripts/ml_tests/tf/resnet/Dockerfile -t tf-dlc-gcsfuse --build-arg DLC_IMAGE_NAME=tf-gpu.2-13 --build-arg BUCKET_TYPE="${BUCKET_TYPE}"

echo "Running the docker image build in the previous step..."
sudo docker run --gpus all --name tf_model_container --privileged -d \
-v $HOME/github/gcsfuse/container_artifacts/logs:/home/logs:rw,rshared \
-v $HOME/github/gcsfuse/container_artifacts/output:/home/output:rw,rshared --shm-size=24g tf-dlc-gcsfuse:latest

# Wait for the script completion as well as logs output.
sudo docker logs -f tf_model_container
