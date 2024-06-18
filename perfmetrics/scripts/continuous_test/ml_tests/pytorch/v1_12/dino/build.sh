#!/bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
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

# This will stop execution when any command will have non-zero status.
set -e

VM_NAME="pytorch-dino-7d"
ZONE_NAME="us-west1-b"
ARTIFACTS_BUCKET_PATH="gs://gcsfuse-ml-tests-logs/ci_artifacts/pytorch/v1_12/dino"
TEST_SCRIPT_PATH="github/gcsfuse/perfmetrics/scripts/ml_tests/pytorch/v1_12/dino/setup_host_and_run_container.sh"
PYTORCH_VERSION="v1_12"

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts/continuous_test/ml_tests/"

source run_and_manage_test.sh $VM_NAME $ZONE_NAME $ARTIFACTS_BUCKET_PATH $TEST_SCRIPT_PATH $PYTORCH_VERSION
