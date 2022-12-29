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

# Image with gcsfuse installed and its package (.deb)
FROM golang:1.18.4 as gcsfuse-package

RUN apt-get update -qq && apt-get install -y ruby ruby-dev rubygems build-essential rpm fuse && gem install --no-document fpm

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GO111MODULE=auto

ARG GCSFUSE_REPO="/run/gcsfuse/"
ADD . ${GCSFUSE_REPO}
WORKDIR ${GCSFUSE_REPO}
RUN go install ./tools/build_gcsfuse

RUN build_gcsfuse . /tmp $(git log -1 --format=format:"%H")

FROM nvcr.io/nvidia/pytorch:22.03-py3

COPY --from=gcsfuse-package /tmp/bin/gcsfuse /usr/bin/gcsfuse
COPY --from=gcsfuse-package /tmp/sbin/mount.gcsfuse /sbin/mount.gcsfuse

# Allow non-root users to specify the allow_other or allow_root mount options
RUN echo "user_allow_other" > /etc/fuse.conf

RUN pip3 install timm

WORKDIR "/pytorch_dino/"

RUN git clone "https://github.com/facebookresearch/dino"

COPY --from=gcsfuse-package /run/gcsfuse/perfmetrics/ml_tests/pytorch_dino_model/setup_container.sh ./
RUN chmod +x setup_container.sh

RUN mkdir -p "gcsfuse_data"

ENTRYPOINT ["/bin/bash", "-c", "./setup_container.sh"]
