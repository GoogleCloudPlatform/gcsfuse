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

# Build an image with gcsfuse packages:
#   > docker build . -t gcsfuse-release --build-arg GCSFUSE_VERSION=0.39.2 --build-arg GCSFUSE_COMMIT_HASH
# Copy the gcsfuse packages to the host:
#   > docker run -it -v /tmp:/output gcsfuse-release cp -r /packages /output

FROM golang:1.18.4 as builder

RUN apt-get update -qq && apt-get install -y ruby ruby-dev rubygems build-essential rpm && gem install --no-document fpm

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GO111MODULE=auto

ARG GCSFUSE_VERSION
ARG GCSFUSE_COMMIT_HASH
ARG GCSFUSE_REPO="github.com/googlecloudplatform/gcsfuse/"
ENV GCSFUSE_PATH "$GOPATH/src/$GCSFUSE_REPO"
RUN go get ${GCSFUSE_REPO}

WORKDIR ${GCSFUSE_PATH}
RUN git checkout "${GCSFUSE_COMMIT_HASH}"

ARG GCSFUSE_BIN="/gcsfuse"
WORKDIR ${GOPATH}
RUN go install ${GCSFUSE_REPO}/tools/build_gcsfuse
RUN mkdir -p ${GCSFUSE_BIN}
RUN build_gcsfuse ${GCSFUSE_PATH} ${GCSFUSE_BIN} ${GCSFUSE_VERSION}
RUN mkdir -p ${GCSFUSE_BIN}/usr && mv ${GCSFUSE_BIN}/bin ${GCSFUSE_BIN}/usr/bin

ARG GCSFUSE_PKG="/packages"
RUN mkdir -p ${GCSFUSE_PKG}
WORKDIR ${GCSFUSE_PKG}
RUN fpm \
    -s dir \
    -t deb \
    -n gcsfuse \
    -C ${GCSFUSE_BIN} \
    -v ${GCSFUSE_VERSION} \
    -d fuse \
    --vendor "" \
    --url "https://$GCSFUSE_REPO" \
    --description "A user-space file system for Google Cloud Storage."
RUN fpm \
    -s dir \
    -t rpm \
    -n gcsfuse \
    -C ${GCSFUSE_BIN} \
    -v ${GCSFUSE_VERSION} \
    -d fuse \
    --vendor "" \
    --url "https://$GCSFUSE_REPO" \
    --description "A user-space file system for Google Cloud Storage."
