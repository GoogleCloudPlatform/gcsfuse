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
# (a) distroless image
#   > docker build ./ --target distroless -t gcsfuse-distroless:v{GCSFUSE_VERSION} --build-arg GCSFUSE_VERSION={GCSFUSE_VERSION}
#   E.g
#   > docker build ./ --target distroless -t gcsfuse-distroless:v0.41.7 --build-arg GCSFUSE_VERSION=0.41.7
# (b) Ubuntu/debian image
#   > docker build . -t gcsfuse-{OS_NAME}:v{GCSFUSE_VERSION} --build-arg GCSFUSE_VERSION={GCSFUSE_VERSION} --build-arg OS_NAME={OS_NAME} --build-arg OS_VERSION={OS_VERSION}
#   E.g.
#   > docker build . -t gcsfuse-ubuntu:v0.41.7 --build-arg GCSFUSE_VERSION=0.41.7 --build-arg OS_NAME=ubuntu --build-arg OS_VERSION=22.04

# Mount bucket to /mnt/gcs on host (/gcs in container):
# (a) using distroless image
#   E.g.
#   > docker run -it --privileged -v $HOME/key.json:/key.json -v /mnt/gcs:/gcs:rw,rshared -e BUCKET_NAME="my-bucket-name" gcsfuse-distroless:v0.41.7
# (b) using ubuntu/debain image
#   E.g.
#   > docker run -it --privileged -v $HOME/key.json:/key.json -v /mnt/gcs:/gcs:rw,rshared -e BUCKET_NAME="my-bucket-name" gcsfuse-ubuntu:v0.41.7

ARG OS_VERSION
ARG OS_NAME

# Image with gcsfuse installed and its package (.deb)
FROM golang:1.22.0 as gcsfuse-package

RUN apt-get update -qq && apt-get install -y ruby ruby-dev rubygems build-essential rpm fuse && gem install --no-document bundler

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GO111MODULE=auto

ARG GCSFUSE_VERSION
ARG GCSFUSE_REPO="github.com/googlecloudplatform/gcsfuse/"
ENV GCSFUSE_PATH "$GOPATH/src/$GCSFUSE_REPO"
RUN go get -d ${GCSFUSE_REPO}

WORKDIR ${GCSFUSE_PATH}
# Branch name for building through a particular branch.
ARG BRANCH_NAME="v${GCSFUSE_VERSION}"
RUN git checkout "${BRANCH_NAME}"

# Install fpm package using bundle
RUN bundle install --gemfile=${GCSFUSE_PATH}/tools/gem_dependency/Gemfile

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

# distroless image with gcsfuse installed.
FROM gcr.io/gke-release/gke-distroless/libc as distroless

# Copy gcsfuse and fuse binaries
ARG GCSFUSE_BIN="/gcsfuse"
COPY --from=gcsfuse-package ${GCSFUSE_BIN}/usr/bin/gcsfuse /bin/
COPY --from=gcsfuse-package ${GCSFUSE_BIN}/sbin/mount.gcsfuse /sbin/
COPY --from=gcsfuse-package /bin/fusermount /bin/
COPY --from=gcsfuse-package /bin/sh /bin/

ENV PATH "/bin:$PATH"
ENV BUCKET_NAME ""

CMD gcsfuse --key-file /key.json -o allow_other --file-mode 777 --dir-mode 777 --foreground --implicit-dirs $BUCKET_NAME /gcs

# Ubuntu/debian image with gcsfuse installed
FROM ${OS_NAME}:${OS_VERSION}

# Copy gcsfuse .deb package from gcsfuse-package image
ARG GCSFUSE_PKG="/packages"
COPY --from=gcsfuse-package ${GCSFUSE_PKG}/*amd64.deb gcsfuse.deb

# Install gcsfuse and dependencies
RUN apt-get update && \
    apt-get install -y ca-certificates fuse && \
    rm -rf /var/lib/apt/lists/* && \
    apt-get clean
RUN dpkg -i gcsfuse.deb && rm gcsfuse.deb

# Allow non-root users to specify the allow_other or allow_root mount options
RUN echo "user_allow_other" > /etc/fuse.conf

ENV BUCKET_NAME ""

CMD gcsfuse --key-file /key.json -o allow_other --file-mode 777 --dir-mode 777 --foreground --implicit-dirs $BUCKET_NAME /gcs
