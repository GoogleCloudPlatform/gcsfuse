#!/bin/bash
#
# Copyright 2024 Google LLC
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

# This script is used purely for automating the run of
# the script run-gke-tests.sh to
# run it periodically as a cron-job.
#
# For your case, add/remove/modify the configuration parameters as you need.
#
# Assumptions for this script to work:
# 1. You have appropriate access to project_id defined below.
# 2. You have the cluster with cluster_name defined below, or enough.
# resources in project_id to create this cluster.

# Print all shell commands.
set -x

# Fail if any command fails.
set -e

# Define configuration parameters.
export project_id=gcs-fuse-test-ml
export project_number=786757290066
export zone=us-west1-b
export cluster_name=gargnitin-gketesting-us-west1b
export node_pool=default-pool
export machine_type=n2-standard-96
export num_nodes=7
export num_ssd=16
export use_custom_csi_driver=true
export output_dir=.
export gcsfuse_branch=garnitin/add-gke-load-testing/v1
export pod_wait_time_in_seconds=300
export pod_timeout_in_seconds=64800
# Pass instance_id from outside to continue previous run, if it got terminated
# somehow (timeout of ssh etc.) 
if test -z ${instance_id}; then
  export instance_id=$(echo ${USER} | sed 's/_google//' | sed 's/_com//')-$(date +%Y%m%d-%H%M%S)
fi
export output_gsheet_id=1UghIdsyarrV1HVNc6lugFZS1jJRumhdiWnPgoEC8Fe4
export output_gsheet_keyfile=gs://gcsfuse-aiml-test-outputs/creds/${project_id}.json
export force_update_gcsfuse_code=true
# Continue previous run if pods had been scheduled/completed already.
test -n ${only_parse} || export only_parse=false

# Create a dedicated folder on the machine.
mkdir -pv ~/gke-testing && cd ~/gke-testing
wget https://raw.githubusercontent.com/googlecloudplatform/gcsfuse/${gcsfuse_branch}/perfmetrics/scripts/testing_on_gke/examples/run-gke-tests.sh -O run-gke-tests.sh
chmod +x run-gke-tests.sh

# Remove previous run's outputs.
rm -rfv log fio/output.csv dlio/output.csv

# Run the script.
start_time=$(date +%Y-%m-%dT%H:%M:%SZ)
echo 'Run started at ${start_time}'
touch log
(./run-gke-tests.sh --debug |& tee -a log) || true
# Use the following if you want to run it in a tmux session instead.
# tmux new-session -d -s ${instance_id} 'bash -c "(./run-gke-tests.sh --debug |& tee -a log); sleep 604800 "'
end_time=$(date +%Y-%m-%dT%H:%M:%SZ)
echo 'Run ended at ${end_time}'

# Some post-run steps to be taken for output collection.
if test -n "${workload_config}"; then
  cp ${workload_config} ./workloads.json
else
  cp src/gcsfuse/perfmetrics/scripts/testing_on_gke/examples/workloads.json .
fi
git -C src/gcsfuse rev-parse HEAD > gcsfuse_commithash
git -C src/gcs-fuse-csi-driver rev-parse HEAD > gcs_fuse_csi_driver_commithash
# Fetch cloud-logs for this run. This has not been tested yet.
# (gcloud logging read --project=${project_id} 'timestamp>="${start_time}"" AND timestamp<="${end_time}" AND resource.labels.cluster_name="${cluster_name}" ' --order=ASC --format=csv\(timestamp\,resource.labels.pod_name,resource.labels.container_name,"text_payload"\) > cloud_logs.txt) &

# Upload outputs to GCS after the run.
output_bucket=gcsfuse-aiml-test-outputs
output_path_uri=gs://${output_bucket}/outputs/${instance_id}
for file in fio/output.csv dlio/output.csv log run-gke-tests.sh workloads.json gcsfuse_commithash gcs_fuse_csi_driver_commithash; do
  if test -f ${file} ; then
    gcloud storage cp --content-type=text/text ${file} ${output_path_uri}/${file}
  fi
done

# Go back to whichever working directory you were in.
cd -
