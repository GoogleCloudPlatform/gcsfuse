#!/bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
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
# pytorch version (e.g. v1_12, v2)
PYTORCH_VESRION=$1
cd "$HOME/github/gcsfuse"
echo "Building docker image containing all pytorch libraries..."
sudo docker build . -f perfmetrics/scripts/ml_tests/pytorch/${PYTORCH_VESRION}/dino/Dockerfile --tag pytorch-gcsfuse

mkdir -p container_artifacts

echo "Running the docker image build in the previous step..."
sudo docker run --gpus all --name=pytorch_automation_container --privileged -d -v $HOME/github/gcsfuse/container_artifacts:/pytorch_dino/run_artifacts:rw,rshared \
--shm-size=128g pytorch-gcsfuse:latest

# Wait for the script completion as well as logs output.
sudo docker logs -f pytorch_automation_container
