#!/bin/bash

# This will stop execution when any command will have non-zero status.
set -e

VM_NAME="pytorch-dino-7d"
ZONE_NAME="us-west1-b"
ARTIFACTS_BUCKET_PATH="gs://gcsfuse-ml-data/ci_artifacts/pytorch/dino"
TEST_SCRIPT_PATH="github/gcsfuse/perfmetrics/scripts/ml_tests/pytorch/dino/setup_host_and_run_model.sh"

cd "$HOME/github/gcsfuse/perfmetrics/scripts/continuous_test/ml_tests/"

chmod +x create_and_manage_test.sh
source create_and_manage_test.sh $VM_NAME $ZONE_NAME $ARTIFACTS_BUCKET_PATH $TEST_SCRIPT_PATH

