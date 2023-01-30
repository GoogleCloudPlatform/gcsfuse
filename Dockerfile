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

# Build an alpine image with gcsfuse installed from the source code:
#  > docker build . -t gcsfuse
# Mount the gcsfuse to /mnt/gcs:
#  > docker run --privileged --device /fuse -v /mnt/gcs:/gcs:rw,rshared gcsfuse

FROM golang:1.19.5-alpine as builder

RUN apk add git

ARG GCSFUSE_REPO="/run/gcsfuse/"
ADD . ${GCSFUSE_REPO}
WORKDIR ${GCSFUSE_REPO}
RUN go install ./tools/build_gcsfuse
RUN build_gcsfuse . /tmp $(git log -1 --format=format:"%H")

FROM alpine:3.13

RUN apk add --update --no-cache bash ca-certificates fuse

COPY --from=builder /tmp/bin/gcsfuse /usr/local/bin/gcsfuse
COPY --from=builder /tmp/sbin/mount.gcsfuse /usr/sbin/mount.gcsfuse
ENTRYPOINT ["gcsfuse", "-o", "allow_other", "--foreground", "--implicit-dirs", "/gcs"]
