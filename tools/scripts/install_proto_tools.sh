#!/usr/bin/env bash
# Copyright 2026 Google LLC
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

set -euo pipefail

PROTOC_VERSION="36.1"
PROTOC_GEN_GO_VERSION="v1.36.12"

# Take the first entry if GOPATH contains multiple colon-delimited paths.
GOPATH_RAW="$(go env GOPATH)"
GOPATH_DIR="${GOPATH_RAW%%:*}"
PROTOC_BIN="${GOPATH_DIR}/bin/protoc"

export PATH="${GOPATH_DIR}/bin:${PATH}"

# 1. Install pinned protoc into GOPATH/bin if missing or version mismatch.
CURRENT_VERSION="$("${PROTOC_BIN}" --version 2>/dev/null || true)"
EXPECTED_VERSION="libprotoc ${PROTOC_VERSION}"

if [[ ! -x "${PROTOC_BIN}" ]] || [[ "${CURRENT_VERSION}" != "${EXPECTED_VERSION}" ]]; then
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "${OS}" in
    darwin) OS="osx" ;;
    linux)  OS="linux" ;;
    *)
      echo "Unsupported operating system: ${OS}" >&2
      exit 1
      ;;
  esac

  ARCH="$(uname -m)"
  case "${ARCH}" in
    x86_64|amd64) ARCH="x86_64" ;;
    arm64|aarch64) ARCH="aarch_64" ;;
    *)
      echo "Unsupported architecture: ${ARCH}" >&2
      exit 1
      ;;
  esac

  ZIP="protoc-${PROTOC_VERSION}-${OS}-${ARCH}.zip"
  URL="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${ZIP}"
  TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "${TMP_DIR}"' EXIT

  echo "Installing protoc v${PROTOC_VERSION} (${OS}-${ARCH}) into ${GOPATH_DIR}/bin..."
  mkdir -p "${GOPATH_DIR}/bin"
  curl -sSL --fail "${URL}" -o "${TMP_DIR}/${ZIP}"

  # Unpack into temp dir first to avoid unzip glob issues and avoid dropping readme.txt into GOPATH.
  unzip -q -o "${TMP_DIR}/${ZIP}" -d "${TMP_DIR}/unpacked"
  cp -f "${TMP_DIR}/unpacked/bin/protoc" "${PROTOC_BIN}"
  cp -rf "${TMP_DIR}/unpacked/include" "${GOPATH_DIR}/"
  chmod +x "${PROTOC_BIN}"
fi

# 2. Install pinned protoc-gen-go into GOPATH/bin.
GOBIN="${GOPATH_DIR}/bin" go install "google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}"
