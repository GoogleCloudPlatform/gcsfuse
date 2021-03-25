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

FROM golang:1.16.2-alpine as builder

RUN apk add git

ENV CGO_ENABLED=0
ENV GOOS=linux

ARG GCSFUSE_VERSION="master"
ARG GCSFUSE_REPO="${GOPATH}/src/github.com/googlecloudplatform/gcsfuse/"
ADD . ${GCSFUSE_REPO}

WORKDIR ${GCSFUSE_REPO}
RUN git checkout ${GCSFUSE_VERSION}
RUN go install ./tools/build_gcsfuse
RUN build_gcsfuse . /tmp ${GCSFUSE_VERSION}

FROM alpine:3.12

RUN apk add --update --no-cache bash ca-certificates fuse

COPY --from=builder /tmp/bin/gcsfuse /usr/local/bin/gcsfuse
COPY --from=builder /tmp/sbin/mount.gcsfuse /usr/sbin/mount.gcsfuse
