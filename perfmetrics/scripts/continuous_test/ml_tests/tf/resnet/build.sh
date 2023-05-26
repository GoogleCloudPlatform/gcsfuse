#!/bin/bash

# This will stop execution when any command will have non-zero status.
set -e

VM_NAME="tf-resnet-7d"
ZONE_NAME="us-central1-c"
ARTIFACTS_BUCKET_PATH="gs://gcsfuse-ml-data/ci_artifacts/tf/resnet"
TEST_SCRIPT_PATH="github/gcsfuse/perfmetrics/scripts/ml_tests/tf/resnet/setup_host_and_run_model.sh"

cd "$HOME/github/gcsfuse/perfmetrics/scripts/continuous_test/ml_tests/"

chmod +x create_and_manage_test.sh
source create_and_manage_test.sh $VM_NAME $ZONE_NAME $ARTIFACTS_BUCKET_PATH $TEST_SCRIPT_PATH

