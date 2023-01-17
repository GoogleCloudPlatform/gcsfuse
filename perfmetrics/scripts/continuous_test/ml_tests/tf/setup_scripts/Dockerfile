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

ARG DLC_IMAGE_NAME

FROM gcr.io/deeplearning-platform-release/${DLC_IMAGE_NAME}

RUN sudo apt-get update
RUN sudo apt-get install -y bash ca-certificates

# Install go-lang
RUN wget -O go_tar.tar.gz https://go.dev/dl/go1.19.4.linux-amd64.tar.gz
RUN sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
RUN export PATH=$PATH:/usr/local/go/bin

# Allow non-root users to specify the allow_other or allow_root mount options
RUN echo "user_allow_other" > /etc/fuse.conf

RUN pip3 install --user tf-models-official==2.10.0

# Fail building the container image if folder.py is not at expected location.
RUN sh -c 'if [ -f "/root/.local/lib/python3.7/site-packages/official/core/train_lib.py" ]; then echo "file exists"; else echo "train_lib.py file not present in expected location. Please correct the location. Exiting"; exit 1; fi'
RUN sh -c 'if [ -f "/root/.local/lib/python3.7/site-packages/orbit/controller.py" ]; then echo "file exists"; else echo "controller.py file not present in expected location. Please correct the location. Exiting"; exit 1; fi'

WORKDIR "/tf_test/"
COPY . .
COPY ./perfmetrics/scripts/continuous_test/ml_tests/tf/setup_scripts/resnet.py ./
COPY ./perfmetrics/scripts/continuous_test/ml_tests/tf/setup_scripts/train_lib.py /root/.local/lib/python3.7/site-packages/official/core/train_lib.py
COPY ./perfmetrics/scripts/continuous_test/ml_tests/tf/setup_scripts/controller.py /root/.local/lib/python3.7/site-packages/orbit/controller.py
COPY ./perfmetrics/scripts/continuous_test/ml_tests/tf/setup_scripts/gcsfuse_mount.sh ./
RUN chmod 777 gcsfuse_mount.sh

RUN mkdir -p "myBucket"
ENTRYPOINT sleep infinity
