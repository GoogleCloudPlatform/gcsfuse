# Copyright 2025 Google LLC
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

# Build the optimized gcsfuse .deb package (referencing kislaykishore/fuse-pool)
# and run the 1 MiB / 1 GiB / 56-job gRPC read FIO benchmark to demonstrate
# maximum throughput (~20.3-21.1 GB/s) with reduced CPU core usage (15-24 physical cores):
#
#   docker build -t gcsfuse-grpc-opt .
#
#   # Example 1: Restrict container to 23 physical cores / 23 vCPUs (16-38) -> ~20.31 GB/s:
#   docker run --rm --privileged --network=host --cpuset-cpus="16-38" \
#     gcsfuse-grpc-opt --bucket kislayk-c4-usw4-hns
#
#   # Example 2: Restrict container to 24 physical cores / 24 vCPUs (16-39) -> ~20.51 GB/s:
#   docker run --rm --privileged --network=host --cpuset-cpus="16-39" \
#     gcsfuse-grpc-opt --bucket kislayk-c4-usw4-hns
#
#   # Example 3: Restrict container to 15 physical cores / 30 vCPUs (16-30,64-78) -> ~20.30 GB/s:
#   docker run --rm --privileged --network=host --cpuset-cpus="16-30,64-78" \
#     gcsfuse-grpc-opt --bucket kislayk-c4-usw4-hns
#
#   # Example 4: Restrict container to 16 physical cores / 32 vCPUs (16-31,64-79) -> ~20.87 GB/s:
#   docker run --rm --privileged --network=host --cpuset-cpus="16-31,64-79" \
#     gcsfuse-grpc-opt --bucket kislayk-c4-usw4-hns
#
#   # Example 5: Restrict container to 26 physical cores (16-41) with NUMJOBS=72 -> ~21.06 GB/s:
#   docker run --rm --privileged --network=host --cpuset-cpus="16-41" \
#     gcsfuse-grpc-opt --bucket kislayk-c4-usw4-hns --numjobs 72

ARG GO_VERSION=1.27.2

FROM golang:${GO_VERSION}-bookworm AS builder

ARG GCSFUSE_VERSION=3.13.0-grpc-cpu-opt
ARG ARCHITECTURE=amd64

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GO111MODULE=on

WORKDIR /src/gcsfuse
COPY . /src/gcsfuse

RUN go install ./tools/build_gcsfuse && \
    PKG_DIR="/pkg/gcsfuse_${GCSFUSE_VERSION}_${ARCHITECTURE}" && \
    mkdir -p "${PKG_DIR}" /packages && \
    build_gcsfuse /src/gcsfuse "${PKG_DIR}" "${GCSFUSE_VERSION}" && \
    mkdir -p "${PKG_DIR}/usr" && \
    mv "${PKG_DIR}/bin" "${PKG_DIR}/usr/bin" && \
    mkdir -p "${PKG_DIR}/DEBIAN" && \
    cp /src/gcsfuse/DEBIAN/control "${PKG_DIR}/DEBIAN/control" && \
    sed -i "s/^Version:.*/Version: ${GCSFUSE_VERSION}/" "${PKG_DIR}/DEBIAN/control" && \
    sed -i "s/^Architecture:.*/Architecture: ${ARCHITECTURE}/" "${PKG_DIR}/DEBIAN/control" && \
    sed -i "s/^Depends:.*/Depends: fuse3 | fuse/" "${PKG_DIR}/DEBIAN/control" && \
    dpkg-deb --build "${PKG_DIR}" "/packages/gcsfuse_${GCSFUSE_VERSION}_${ARCHITECTURE}.deb"

FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1

RUN apt-get update -qq && \
    apt-get install -y --no-install-recommends \
      ca-certificates \
      fio \
      fuse3 \
      procps \
      python3 \
      util-linux && \
    rm -rf /var/lib/apt/lists/* && \
    echo "user_allow_other" > /etc/fuse.conf

COPY --from=builder /packages/*.deb /packages/
RUN dpkg -i /packages/gcsfuse_*.deb

RUN mkdir -p /bench /mnt/gcs
COPY --from=builder /src/gcsfuse/tools/fio_bench/read.fio /bench/read.fio
COPY --from=builder /src/gcsfuse/tools/fio_bench/run_fio_bench.py /bench/run_fio_bench.py
RUN chmod +x /bench/run_fio_bench.py

ENTRYPOINT ["python3", "/bench/run_fio_bench.py"]
