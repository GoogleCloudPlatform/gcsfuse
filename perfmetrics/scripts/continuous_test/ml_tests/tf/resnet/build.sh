#!/bin/bash

# This will stop execution when any command will have non-zero status.
set -e

VM_NAME="tf-resnet-7d"
ZONE_NAME="us-west1-b"
ARTIFACTS_BUCKET_PATH="gs://gcsfuse-ml-tests-logs/ci_artifacts/tf/resnet"
TEST_SCRIPT_PATH="github/gcsfuse/perfmetrics/scripts/ml_tests/tf/resnet/setup_host_and_run_model.sh"

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts/continuous_test/ml_tests/"

chmod +x run_and_manage_test.sh
source run_and_manage_test.sh $VM_NAME $ZONE_NAME $ARTIFACTS_BUCKET_PATH $TEST_SCRIPT_PATH

