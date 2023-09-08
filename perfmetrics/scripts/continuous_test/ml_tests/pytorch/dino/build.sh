#!/bin/bash

# This will stop execution when any command will have non-zero status.
set -e

VM_NAME="test-pytorch-dino-7d"
ZONE_NAME="asia-northeast1-c"
ARTIFACTS_BUCKET_PATH="gs://ashmeen-test-ml/ci_artifacts/pytorch/dino"
TEST_SCRIPT_PATH="github/gcsfuse/perfmetrics/scripts/ml_tests/pytorch/dino/setup_host_and_run_model.sh"

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts/continuous_test/ml_tests/"

chmod +x run_and_manage_test.sh
source run_and_manage_test.sh $VM_NAME $ZONE_NAME $ARTIFACTS_BUCKET_PATH $TEST_SCRIPT_PATH

