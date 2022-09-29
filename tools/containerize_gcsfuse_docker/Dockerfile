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

# Build image with gcsfuse installed:
#   > docker build . -t gcsfuse-{OS_NAME}:v{GCSFUSE_VERSION} --build-arg GCSFUSE_VERSION={GCSFUSE_VERSION} --build-arg OS_NAME={OS_NAME} --build-arg OS_VERSION={OS_VERSION} --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
#   E.g.
#   > docker build . -t gcsfuse-ubuntu:v0.41.7 --build-arg GCSFUSE_VERSION=0.41.7 --build-arg OS_NAME=ubuntu --build-arg OS_VERSION=22.04 --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')

# Mount bucket in container using gcsfuse:
# E.g.
#   > docker run --privileged -it -v $HOME/key.json:$HOME/key.json gcsfuse-ubuntu:v0.41.7 gcsfuse --key-file $HOME/key.json --foreground bucket-name /tmp

ARG OS_NAME
ARG OS_VERSION
FROM ${OS_NAME}:${OS_VERSION}

# Install dependencies
ARG GCSFUSE_VERSION
ARG GCSFUSE_PACKAGE_NAME=gcsfuse_${GCSFUSE_VERSION}_amd64
RUN apt-get update && apt-get install -y ca-certificates fuse curl && rm -rf /var/lib/apt/lists/* && apt-get clean

# Download and install gcsfuse package
RUN curl -L -O https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v${GCSFUSE_VERSION}/${GCSFUSE_PACKAGE_NAME}.deb
RUN dpkg -i ${GCSFUSE_PACKAGE_NAME}.deb && rm ${GCSFUSE_PACKAGE_NAME}.deb

# Labels.
ARG BUILD_DATE
LABEL build-date=$BUILD_DATE
LABEL name="GoogleCloudPlatform/gcsfuse"
LABEL version="v${GCSFUSE_VERSION}"
LABEL description="A user-space file system for Google Cloud Storage."
LABEL url="https://cloud.google.com/storage/docs/gcs-fuse"
LABEL vcs-url="https://github.com/GoogleCloudPlatform/gcsfuse"

CMD ["gcsfuse", "--help"]
