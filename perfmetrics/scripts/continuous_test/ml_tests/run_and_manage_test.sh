#!/bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This will stop execution when any command will have non-zero status.
set -e

# 7.5 days of timeout for running test
TIMEOUT=$(echo "7.5*24*60*60" | bc)
GCP_PROJECT="gcs-fuse-test"
# Name of test VM.
VM_NAME=$1
# Zone of test VM.
ZONE_NAME=$2
# Bucket path where the test VM artifacts should be saved.
ARTIFACTS_BUCKET_PATH=$3
# Path of test script relative to $HOME inside test VM.
TEST_SCRIPT_PATH=$4

function delete_existing_vm_and_create_new () {
  (
    set +e
    echo "Deleting VM $VM_NAME in zone $ZONE_NAME."
    sudo gcloud compute instances delete $VM_NAME --zone $ZONE_NAME --quiet
    if [ $? -eq 0 ];
    then
      echo "Machine deleted successfully !"
    else
      echo "Machine was not deleted as it doesn't exist."
    fi
  )

  echo "Wait for 30 seconds for old VM to be deleted"
  sleep 30s

  echo "Creating VM $VM_NAME in zone $ZONE_NAME"
  sudo gcloud compute instances create $VM_NAME \
      --project=$GCP_PROJECT \
      --zone=$ZONE_NAME \
      --machine-type=a2-highgpu-2g \
      --network-interface=network-tier=PREMIUM,nic-type=GVNIC,stack-type=IPV4_ONLY,subnet=default \
      --metadata=enable-oslogin=true \
      --maintenance-policy=TERMINATE \
      --provisioning-model=STANDARD \
      --service-account=927584127901-compute@developer.gserviceaccount.com \
      --scopes=https://www.googleapis.com/auth/cloud-platform \
      --accelerator=count=2,type=nvidia-tesla-a100 \
      --create-disk=auto-delete=yes,boot=yes,device-name=$VM_NAME,image=projects/ubuntu-os-cloud/global/images/ubuntu-2004-focal-v20230523,mode=rw,size=200,type=projects/$GCP_PROJECT/zones/us-central1-c/diskTypes/pd-balanced \
      --no-shielded-secure-boot \
      --shielded-vtpm \
      --shielded-integrity-monitoring \
      --labels=goog-ec-src=vm_add-gcloud \
      --reservation-affinity=any

  echo "Wait for 30 seconds for new VM to be initialised"
  sleep 30s

  echo "Delete existing ssh keys "
  # This is required to avoid issue: https://github.com/kyma-project/test-infra/issues/93
  for i in $(sudo gcloud compute os-login ssh-keys list | grep -v FINGERPRINT); do sudo gcloud compute os-login ssh-keys remove --key $i; done

  # Requires running first ssh command with --quiet option to initialize keys.
  # Otherwise it prompts for yes and no.
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --quiet --command "echo 'Running from VM'"
}

# Takes commit id of on-going test run ($1) artifacts to GCS bucket
function copy_run_artifacts_to_gcs () {
  (
    # We don't want to exit if failure occurs while copying GCSFuse logs because
    # gsutil always gives error (even the files are copied) while uploading
    # files that are changing while uploading and gcsfuse logs are changing
    # when the test is running.
    set +e
    echo "Copying GCSFuse logs to GCS bucket"
    sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "gsutil rsync -R -d \$HOME/github/gcsfuse/container_artifacts/ $ARTIFACTS_BUCKET_PATH/$1/container_artifacts"
  )
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "gsutil cp \$HOME/build.out $ARTIFACTS_BUCKET_PATH/$1/build.out"
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "gsutil cp \$HOME/build.err $ARTIFACTS_BUCKET_PATH/$1/build.err"
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "gsutil cp $ARTIFACTS_BUCKET_PATH/status.txt $ARTIFACTS_BUCKET_PATH/$1/"
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "gsutil cp $ARTIFACTS_BUCKET_PATH/commit.txt $ARTIFACTS_BUCKET_PATH/$1/"
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "gsutil cp $ARTIFACTS_BUCKET_PATH/start_time.txt $ARTIFACTS_BUCKET_PATH/$1/"
  echo "Build logs from test VM copied to GCS bucket for the run $1"
}

function cat_run_artifacts () {
  echo "Below is the stdout of build on test VM"
  gsutil cat $ARTIFACTS_BUCKET_PATH/$1/build.out

  echo "Below is the stderr of build on test VM"
  gsutil cat $ARTIFACTS_BUCKET_PATH/$1/build.err
}

# Returns status of on-going test run.
function get_run_status () {
  status=$(gsutil cat $ARTIFACTS_BUCKET_PATH/status.txt)
  echo $status
}

# Returns commit id of on-going test run.
function get_run_commit_id () {
  commit_id=$(gsutil cat $ARTIFACTS_BUCKET_PATH/commit.txt)
  echo $commit_id
}

sudo gcloud config set project $GCP_PROJECT
current_status=$(get_run_status)
echo "The current status is $current_status"
exit_status=0

# Transitions:
# START to START: If model run is not triggerred due to some error.
# START to RUNNING: If model is successfully triggerred on GPU. This state is 
#                   changed by setup_host.sh that runs inside docker container of test VM.
if [ $current_status == "START" ];
then
  echo "Update commit Id for the run"
  commit_id=$(git rev-parse HEAD)
  echo $commit_id > commit.txt
  gsutil cp commit.txt $ARTIFACTS_BUCKET_PATH/

  delete_existing_vm_and_create_new
  
  echo "Clone the gcsfuse repo on test VM"
  # To-do: change checkout to master branch before merging.
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "mkdir github; cd github; git clone https://github.com/GoogleCloudPlatform/gcsfuse.git; cd gcsfuse; git checkout ai_ml_tests;"
  echo "Trigger the build script on test VM"
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "bash \$HOME/$TEST_SCRIPT_PATH 1> ~/build.out 2> ~/build.err &"
  echo "Wait for 10 minutes for test VM to setup for test and to change the status from START to RUNNING."
  sleep 600s

  # If the model is still not running after waiting, then the build should fail.
  if [ $(get_run_status) != "RUNNING" ];
  then
    echo "The model has not started."
    exit_status=1
  fi
# If the current state is running, then check for timeout. If timed out then the
# build should fail.
# Transitions: RUNNING TO ERROR: If the model fails.
#              RUNNING TO COMPLETE: If the model succeeds.
# The above transitions are done by docker container running inside test VM.
elif [ $current_status == "RUNNING" ];
then
  # Check for timeout.
  start_time=$(gsutil cat $ARTIFACTS_BUCKET_PATH/start_time.txt)
  current_time=$(date +"%s")
  time_elapsed=$(expr $current_time - $start_time)
  if (( $(echo "$time_elapsed > $TIMEOUT" | bc -l) ));
  then
    echo "The tests have time out, start_time was $start_time, current time is $current_time, time elapsed is $time_elapsed"
    exit_status=1
  fi
# Fail the build if the current state is ERROR. This state is set by docker
# container running inside test VM if model fails.
# Transitions: ERROR TO START: This has to be changed manually when the model/
#              error is fixed.
elif [ $current_status == "ERROR" ];
then
  exit_status=1
# Transitions: COMPLETE TO START: Once the current run is complete, mark the
#              state as START.
# The status "COMPLETE" is set by docker container inside test VM when the model
# is successfully trained.
elif [ $current_status == "COMPLETE" ];
then
  exit_status=0
  # change status back to start
  echo "START" > status.txt
  gsutil cp status.txt $ARTIFACTS_BUCKET_PATH/
else
  echo "Unknown state in status file. Please check."
  exit 1
fi

commit_id=$(get_run_commit_id)
copy_run_artifacts_to_gcs $commit_id
cat_run_artifacts $commit_id

exit $exit_status