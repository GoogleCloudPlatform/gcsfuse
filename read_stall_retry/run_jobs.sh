#!/bin/bash

# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Make sure that job.yaml is in the same directory from where this script is executed

set -x

function run_job {
    NAME=""  # Initialize job_name

    if [[ "$DATA_BUCKET_NAME" == "vipinydv-us-central1" ]]; then
        NAME="fastenvironment"
    else
        NAME="slowenvironment"
    fi

    if [[ "$READ_STALL_RETRY" == "true" ]]; then
        NAME+="-readstall"
    fi

    if [[ ! "${FILE_CACHE_CAPACITY:0:1}" == "0" ]]; then  # Check if it DOESN'T start with 0
        if [[ "$PARALLEL_DOWNLOAD" == "true" ]]; then
            NAME+="-paralleldownload"
        else
            NAME+="-filecache"
        fi
    else
        NAME+="-genericread"
    fi
    export NAME
    envsubst '$DATA_BUCKET_NAME $READ_STALL_RETRY $FILE_CACHE_CAPACITY $PARALLEL_DOWNLOAD $NAME' < job.yaml | kubectl apply -f -

    # Wait for the specific job to complete or fail
    kubectl wait --for=condition=complete --timeout=2h job/"$NAME"

    # Check the exit code of the wait command
    wait_result=$?

    # Delete ONLY the specific job
    if [[ $wait_result -eq 0 ]]; then
        echo "Job '$NAME' completed successfully."
    elif [[ $wait_result -ne 0 ]]; then
        echo "Job '$NAME' failed or aborted due to timeout."
    fi

    kubectl delete job/"$NAME"  # Delete the specific job

    sleep 30
}

# Define arrays
DATA_BUCKET_NAMES=("vipinydv-us-central1" "vipinydv-asia-south1")
READ_STALL_RETRIES=(false true)
FILE_CACHE_CAPACITIES=("0Ti" "4Ti" "4Ti")
PARALLEL_DOWNLOADS=(false false true)

# Loop through all combinations
for DATA_BUCKET_NAME in "${DATA_BUCKET_NAMES[@]}"; do
    for READ_STALL_RETRY in "${READ_STALL_RETRIES[@]}"; do
        for idx in "${!FILE_CACHE_CAPACITIES[@]}"; do
                export DATA_BUCKET_NAME
                export READ_STALL_RETRY
                export FILE_CACHE_CAPACITY="${FILE_CACHE_CAPACITIES[$idx]}"
                export PARALLEL_DOWNLOAD="${PARALLEL_DOWNLOADS[$idx]}"
                run_job
        done
    done
done
