#!/bin/bash

# This will stop execution when any command will have non-zero status.
set -e

# Timeout for running test
TIMEOUT=$(echo "7.5*24*60*60" | bc)
GCP_PROJECT="gcs-fuse-test"
VM_NAME=$1
ZONE_NAME=$2
ARTIFACTS_BUCKET_PATH=$3
TEST_SCRIPT_PATH=$4

# Helper Functions
function delete_existing_vm_and_create_new () {
  (
    echo "Deleting VM $VM_NAME"
    sudo gcloud compute instances delete $VM_NAME --zone $ZONE_NAME --quiet
  ) || (
    if [ $? != 0 ];
    then
      echo "Machine was not deleted as it doesn't exist."
    fi
  )

  echo "Wait for 60 seconds for old VM to be deleted"
  sleep 30s

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

  echo "Wait for 60 seconds for new VM to be initialised"
  sleep 30s
}

function copy_artifacts_to_gcs () {
  (
    sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "gsutil rsync -R \$HOME/github/gcsfuse/container_artifacts/ $ARTIFACTS_BUCKET_PATH/$1/container_artifacts"
  ) || (
    if [ $? -eq 0 ]
    then
        echo "GCSFuse logs successfully copied to GCS bucket $ARTIFACTS_BUCKET_PATH"
    else
        echo "GCSFuse logs are not copied for the run $1"
    fi
  )
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "gsutil cp \$HOME/build.out $ARTIFACTS_BUCKET_PATH/$1/build.out"
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "gsutil cp \$HOME/build.err $ARTIFACTS_BUCKET_PATH/$1/build.err"
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "gsutil cp $ARTIFACTS_BUCKET_PATH/status.txt $ARTIFACTS_BUCKET_PATH/$1/"
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "gsutil cp $ARTIFACTS_BUCKET_PATH/commit.txt $ARTIFACTS_BUCKET_PATH/$1/"
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "gsutil cp $ARTIFACTS_BUCKET_PATH/start_time.txt $ARTIFACTS_BUCKET_PATH/$1/"
  echo "Build logs copied to GCS for the run $1"
}

function get_run_status () {
  status=$(gsutil cat $ARTIFACTS_BUCKET_PATH/status.txt)
  echo $status
}

function get_commit_id () {
  commit_id=$(gsutil cat $ARTIFACTS_BUCKET_PATH/commit.txt)
  echo $commit_id
}

# Set project
sudo gcloud config set project $GCP_PROJECT
exit_status=0

# Transitions:
# START to START: If model doesn't run due to some error.
# START to RUNNING: If model is successfully triggerred on GPU. This state is 
#                   changed by setup_host.sh that runs inside docker container.
if [ $(get_run_status) == "START" ];
then
  delete_existing_vm_and_create_new
  
  echo "Clone the gcsfuse repo on VM (GPU)"
  # Requires running first ssh command with --quiet option to initialize keys.
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --quiet --command "echo 'Running from VM'"
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "mkdir github; cd github; git clone https://github.com/GoogleCloudPlatform/gcsfuse.git; cd gcsfuse; git checkout ai_ml_tests;"
  echo "Trigger the build script on VM (GPU)"
  sudo gcloud compute ssh $VM_NAME --zone $ZONE_NAME --internal-ip --command "bash \$HOME/$TEST_SCRIPT_PATH 1> ~/build.out 2> ~/build.err &"
  echo "Wait for 10 minutes for VM (GPU) to setup for test !"
  sleep 600s

  echo "Update commit Id for the run"
  commit_id=$(git rev-parse HEAD)
  echo $commit_id > commit.txt
  gsutil cp commit.txt $ARTIFACTS_BUCKET_PATH/
  # If the model is still not running after waiting then the build should fail.
  if [ $(get_run_status) != "RUNNING" ];
  then
    echo "The model has not started."
    exit_status=1
  fi
# If the current state is running, then check for timeout. If timedout then the
# build should fail.
# Transitions: RUNNING TO ERROR: If the model fails.
#              RUNNING TO COMPLETE: If the model succeeds.
elif [ $(get_run_status) == "RUNNING" ];
then
  start_time=$(gsutil cat $ARTIFACTS_BUCKET_PATH/start_time.txt)
  current_time=$(date +"%s")
  time_elapsed=$(expr $current_time - $start_time)
  if (( $(echo "$time_elapsed > $TIMEOUT" | bc -l) ));
  then
    echo "The tests have time out, start_time was $start_time, current time is $current_time, time elapsed is $time_elapsed"
    exit_status=1
  fi
# fail the build if the current state is ERROR. This state is set by model.
# Transitions: ERROR TO START: This has to be changed manually when the model/
#              error is fixed.
elif [ $(get_run_status) == "ERROR" ];
then
  exit_status=1
# Transitions: COMPLETE TO START: Once the current run is complete, mark the
#              state as START.
elif [ $(get_run_status) == "COMPLETE" ];
then
  exit_status=0
  # change status back to start
  echo "START" > status.txt
  gsutil cp status.txt $ARTIFACTS_BUCKET_PATH/
else
  echo "Unknown state in status file. Please check."
  exit 1

fi

commit_id=$(get_commit_id)
copy_artifacts_to_gcs $commit_id

echo "Below is the stdout of build on VM (GPU)"
gsutil cat $ARTIFACTS_BUCKET_PATH/$commit_id/build.out

echo "Below is the stderr of build on VM (GPU)"
gsutil cat $ARTIFACTS_BUCKET_PATH/$commit_id/build.err

exit $exit_status