# Copyright 2020 Google Inc. All Rights Reserved.
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

# This dockerfile contains deep learning container as base image
# and is used for running tf based resnet18 model on the gpu vms for 
# kokoro test

ARG DLC_IMAGE_NAME

FROM gcr.io/deeplearning-platform-release/${DLC_IMAGE_NAME}

RUN sudo apt-get update
RUN sudo apt-get install -y bash ca-certificates

# Allow non-root users to specify the allow_other or allow_root mount options
RUN echo "user_allow_other" > /etc/fuse.conf

WORKDIR "/tf_test/"

COPY ./perfmetrics/scripts/continuous_test/ml_tests/tf/setup_scripts/setup_container.sh .
COPY ./perfmetrics/scripts/continuous_test/ml_tests/tf/setup_scripts/resnet_runner.py .
RUN chmod +x setup_container.sh

RUN mkdir -p "myBucket"
ENTRYPOINT ["/bin/bash", "-c", "./setup_container.sh"]
