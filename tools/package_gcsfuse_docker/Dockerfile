# Copyright 2020 Google LLC
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

# This Dockerfile builds gcsfuse DEB and RPM packages for different linux architectures.
#
# Build Arguments:
#   GCSFUSE_VERSION  - Version string for the package (required, e.g., "0.39.2")
#   BRANCH_NAME      - Git commit, branch or tag to build from (default: "v${GCSFUSE_VERSION}")
#   ARCHITECTURE     - Target architecture: "amd64" or "arm64" (default: "amd64")
#   GCSFUSE_REPO     - Repository URL (default: https://github.com/googlecloudplatform/gcsfuse/)
#
# Examples:
#
#   Build from version tag (standard release):
#     docker build . -t gcsfuse-release --build-arg GCSFUSE_VERSION=0.39.2
#
#   Build from specific branch/commit/tag (e.g., for testing unreleased changes):
#     docker build . -t gcsfuse-release \
#       --build-arg GCSFUSE_VERSION=0.40.0 \
#       --build-arg BRANCH_NAME=<commit_hash_or_branch_name>
#
#   Build for ARM64 architecture:
#     docker build . -t gcsfuse-release \
#       --build-arg GCSFUSE_VERSION=0.39.2 \
#       --build-arg ARCHITECTURE=arm64
#
#   Extract packages from the container to host:
#     docker run -it -v /tmp:/output gcsfuse-release cp -r /packages /output
#
# Output:
#   - /packages/gcsfuse_{VERSION}_{ARCH}.deb
#   - /packages/gcsfuse-{VERSION}-1.x86_64.rpm (or aarch64 for arm64)

ARG GO_VERSION=1.24.5

FROM golang:${GO_VERSION}-bookworm AS builder

# Build arguments
ARG ARCHITECTURE="amd64"
# GCSFUSE_VERSION: The release version string (e.g. "2.5.4") embedded in package filenames, control files, changelogs, and RPM specs.
ARG GCSFUSE_VERSION
ARG GCSFUSE_REPO="https://github.com/googlecloudplatform/gcsfuse/"
# BRANCH_NAME: The Git commit hash, branch, or tag used to build gcsfuse. Defaults to "v${GCSFUSE_VERSION}".
ARG BRANCH_NAME="v${GCSFUSE_VERSION}"
ARG GCSFUSE_PATH="/go/gcsfuse"
ARG GCSFUSE_BIN="/gcsfuse_${GCSFUSE_VERSION}_${ARCHITECTURE}"
ARG GCSFUSE_DOC="${GCSFUSE_BIN}/usr/share/doc/gcsfuse"
ARG GCSFUSE_PKG="/packages"

# Environment variables
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GO111MODULE=auto
ENV PATH="/usr/local/go/bin:${PATH}"

RUN apt-get update -qq && apt-get install -y ruby ruby-dev rubygems build-essential && gem install --no-document bundler -v "2.4.12"
RUN apt-get install -y rpm wget

RUN if [ "${ARCHITECTURE}" != "amd64" ] && [ "${ARCHITECTURE}" != "arm64" ]; then \
    echo "Architecture should be amd64 or arm64"; \
    exit 1; \
    fi

RUN git clone ${GCSFUSE_REPO} && \
    cd gcsfuse && \
    git checkout ${BRANCH_NAME}
RUN echo "Building from branch/commit/tag: ${BRANCH_NAME}"

# Extract required Go version, determine download URL, and install Go
RUN set -e; \
    # Step 1: Extract Go version from .go-version or go.mod \
    if [ -f gcsfuse/.go-version ]; then \
        GO_VERSION=$(cat gcsfuse/.go-version); \
    else \
        GO_VERSION=$(sed -n 's/^go //p' gcsfuse/go.mod); \
    fi; \
    \
    # Step 2: Determine Go download URL \
    # Starting with Go 1.21, archives on go.dev/dl require 3-part versions (e.g. 1.21.0 vs 1.20) \
    case "$GO_VERSION" in \
        *.*.*) ;; \
        *.*) \
            MINOR=$(echo "$GO_VERSION" | cut -d. -f2); \
            if [ "$MINOR" -ge 21 ] 2>/dev/null; then \
                GO_VERSION="${GO_VERSION}.0"; \
            fi \
            ;; \
    esac; \
    GO_URL="https://go.dev/dl/go${GO_VERSION}.linux-${ARCHITECTURE}.tar.gz"; \
    \
    # Step 3: Download and install Go \
    echo "Downloading and installing Go from ${GO_URL}..." && \
    rm -rf /usr/local/go && \
    wget -q -O go_tar.tar.gz "${GO_URL}" && \
    tar -C /usr/local -xzf go_tar.tar.gz && \
    rm -f go_tar.tar.gz

RUN go version

WORKDIR ${GCSFUSE_PATH}

# Install fpm package using bundle
RUN bundle install --gemfile=${GCSFUSE_PATH}/tools/gem_dependency/Gemfile

WORKDIR ${GCSFUSE_PATH}/tools/build_gcsfuse
RUN go install
RUN mkdir -p ${GCSFUSE_BIN}
RUN build_gcsfuse ${GCSFUSE_PATH} ${GCSFUSE_BIN} ${GCSFUSE_VERSION}
RUN mkdir -p ${GCSFUSE_BIN}/usr && mv ${GCSFUSE_BIN}/bin ${GCSFUSE_BIN}/usr/bin

# Creating structure for debian package as we are using 'dpkg-deb --build' to create debian package
RUN mkdir -p ${GCSFUSE_BIN}/DEBIAN && cp ${GCSFUSE_PATH}/DEBIAN/* ${GCSFUSE_BIN}/DEBIAN/
RUN mkdir -p ${GCSFUSE_DOC}
RUN mv ${GCSFUSE_BIN}/DEBIAN/copyright ${GCSFUSE_DOC} &&  \
    mv ${GCSFUSE_BIN}/DEBIAN/changelog ${GCSFUSE_DOC} && \
    mv ${GCSFUSE_BIN}/DEBIAN/gcsfuse-docs.docs ${GCSFUSE_DOC}
# Update gcsfuse version in changelog and control file
RUN sed -i "1s/.*/gcsfuse (${GCSFUSE_VERSION}) stable; urgency=medium/" ${GCSFUSE_DOC}/changelog && \
    sed -i "1s/.*/Version: ${GCSFUSE_VERSION}/" ${GCSFUSE_BIN}/DEBIAN/control

# Update ARCHITECTURE in control file.
RUN sed -i "6s/.*/Architecture: ${ARCHITECTURE}/" ${GCSFUSE_BIN}/DEBIAN/control

# Compress changelog as required by lintian
RUN gzip -9 -n ${GCSFUSE_DOC}/changelog
# Strip unneeded from binaries as required by lintian
RUN strip --strip-unneeded ${GCSFUSE_BIN}/usr/bin/gcsfuse && \
    strip --strip-unneeded ${GCSFUSE_BIN}/sbin/mount.gcsfuse

RUN mkdir -p ${GCSFUSE_PKG}
WORKDIR ${GCSFUSE_PKG}
# Build the package
RUN dpkg-deb --build ${GCSFUSE_BIN}
RUN mv ${GCSFUSE_BIN}.deb .
RUN fpm \
    -s dir \
    -t rpm \
    -n gcsfuse \
    -C ${GCSFUSE_BIN} \
    -v ${GCSFUSE_VERSION} \
    -d fuse \
    --exclude DEBIAN \
    --rpm-digest sha256 \
    --license Apache-2.0 \
    --vendor "" \
    --url "$GCSFUSE_REPO" \
    --description "A user-space file system for Google Cloud Storage."
