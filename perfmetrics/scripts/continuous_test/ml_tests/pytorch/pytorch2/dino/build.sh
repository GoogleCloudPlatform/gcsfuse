#!/bin/bash

# This will stop execution when any command will have non-zero status.
set -e

VM_NAME="pytorch2-dino-7d"
ZONE_NAME="us-west1-a"
ARTIFACTS_BUCKET_PATH="gs://gcsfuse-ml-tests-logs/ci_artifacts/pytorch/pytorch2/dino"
TEST_SCRIPT_PATH="github/gcsfuse/perfmetrics/scripts/ml_tests/pytorch/pytorch2/dino/setup_host_and_run_model.sh"
PYTORCH_2="pytorch2"

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts/continuous_test/ml_tests/"

source run_and_manage_test.sh $VM_NAME $ZONE_NAME $ARTIFACTS_BUCKET_PATH $TEST_SCRIPT_PATH $PYTORCH_2
