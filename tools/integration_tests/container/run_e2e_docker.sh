#!/bin/bash
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

set -e

# Define image name
IMAGE_NAME="gcsfuse-e2e-tests"
DOCKERFILE_PATH="tools/integration_tests/container/Dockerfile"
REPO_ROOT="$(git rev-parse --show-toplevel)"

# Check if docker is installed
if ! command -v docker &> /dev/null; then
    echo "Error: docker is not installed."
    exit 1
fi

echo "Building Docker image: $IMAGE_NAME..."
docker build -t "$IMAGE_NAME" -f "$DOCKERFILE_PATH" "$REPO_ROOT"

echo "Running E2E tests in container..."

# Determine GCP credentials path
# Default to standard gcloud config location
GCLOUD_CONFIG_DIR="$HOME/.config/gcloud"
MOUNT_GCLOUD_DIR="/root/.config/gcloud"

if [ ! -d "$GCLOUD_CONFIG_DIR" ]; then
    echo "Warning: ~/.config/gcloud directory not found. Authentication may fail."
    echo "Please ensure you are logged in via 'gcloud auth login' and 'gcloud auth application-default login'."
fi

# Run the container
# --privileged: Required for FUSE
# --rm: Clean up container after exit
# -v: Mount credentials
# Pass all arguments to the script inside the container
docker run --privileged --rm \
    -v "$GCLOUD_CONFIG_DIR:$MOUNT_GCLOUD_DIR" \
    -v "$REPO_ROOT:/workspace_mount" \
    --env GOOGLE_APPLICATION_CREDENTIALS="/root/.config/gcloud/application_default_credentials.json" \
    "$IMAGE_NAME" \
    "$@"

# Note: We mount the repo as /workspace_mount for reference if needed,
# but the Dockerfile copies the repo into /workspace.
# If the user wants to test *local changes* without rebuilding,
# we should probably mount the local repo over /workspace.
# The user said "Include the source code at build time" in step 2.
# But "make the tests independent of the environment".
# If I mount the local directory over /workspace, I don't need to rebuild for code changes, but I do need to rebuild for dependency changes.
# The user's requirement "Include the source code at build time" suggests they prefer the copy behavior.
# However, for a developer workflow, mounting is usually preferred.
# I'll stick to the "build time" requirement for now as requested.
