#!/bin/bash

# This will stop execution when any command will have non-zero status.
set -e

TIMEOUT=$(echo "7.5*24*60*60" | bc)

function delete_existing_vm_and_create_new () {
  (
    echo "Deleting VM $1"
    gcloud compute instances delete $1 --zone us-central1-c --quiet
  ) || (
    if [$? != 0]
    then
      echo "Machine was not deleted as it doesn't exist."
    fi
  )

  echo "Wait for 60 seconds for old VM to be deleted"
  sleep 60s

  gcloud compute instances create $1 \
      --project=gcs-fuse-test \
      --zone=us-central1-c \
      --machine-type=a2-highgpu-2g \
      --network-interface=network-tier=PREMIUM,nic-type=GVNIC,stack-type=IPV4_ONLY,subnet=default \
      --metadata=enable-oslogin=true \
      --maintenance-policy=TERMINATE \
      --provisioning-model=STANDARD \
      --service-account=927584127901-compute@developer.gserviceaccount.com \
      --scopes=https://www.googleapis.com/auth/cloud-platform \
      --accelerator=count=2,type=nvidia-tesla-a100 \
      --create-disk=auto-delete=yes,boot=yes,device-name=tf-resnet-7d,image=projects/ubuntu-os-cloud/global/images/ubuntu-2004-focal-v20230523,mode=rw,size=200,type=projects/gcs-fuse-test/zones/us-central1-c/diskTypes/pd-balanced \
      --no-shielded-secure-boot \
      --shielded-vtpm \
      --shielded-integrity-monitoring \
      --labels=goog-ec-src=vm_add-gcloud \
      --reservation-affinity=any

  echo "Wait for 60 seconds for new VM to be initialised"
  sleep 60s
}

function copy_artifacts_to_gcs () {
  (
    gcloud compute ssh $1 --zone us-central1-c --internal-ip --command "gsutil cp -R \$HOME/github/gcsfuse/container_artifacts/ gs://gcsfuse-ml-data/ci_artifacts/tf/resnet/$2"
  ) || (
    if [ $? -eq 0 ]
    then
        echo "GCSFuse logs successfully copied to GCS bucket gcsfuse-ml-data"
    else
        echo "GCSFuse logs are not copied for the run $2"
    fi
  )
  gcloud compute ssh $1 --zone us-central1-c --internal-ip --command "gsutil cp \$HOME/build.out gs://gcsfuse-ml-data/ci_artifacts/tf/resnet/$2/build.out"
  gcloud compute ssh $1 --zone us-central1-c --internal-ip --command "gsutil cp \$HOME/build.err gs://gcsfuse-ml-data/ci_artifacts/tf/resnet/$2/build.err"
  echo "Build logs copied to GCS for the run $2"
}

function get_run_status () {
  status=$(gsutil cat gs://gcsfuse-ml-data/ci_artifacts/tf/resnet/status.txt)
  echo $status
}

function get_commit_id () {
  commit_id=$(gsutil cat gs://gcsfuse-ml-data/ci_artifacts/tf/resnet/commit.txt)
  echo $commit_id
}

gcloud config set project gcs-fuse-test
exit_status=0

if [ $(get_run_status) == "START" ];
then
  delete_existing_vm_and_create_new "tf-resnet-7d"
  echo "Clone the gcsfuse repo on VM (GPU)"
  gcloud compute ssh tf-resnet-7d --zone us-central1-c --internal-ip --quiet --command "echo 'Running from VM'"
  gcloud compute ssh tf-resnet-7d --zone us-central1-c --internal-ip --command "mkdir github; cd github; git clone https://github.com/GoogleCloudPlatform/gcsfuse.git; cd gcsfuse; git checkout ai_ml_tests;"
  echo "Trigger the build script on VM (GPU)"
  gcloud compute ssh tf-resnet-7d --zone us-central1-c --internal-ip --command "cd github/gcsfuse/perfmetrics/scripts/continuous_test/ml_tests/tf/resnet/; export KOKORO_ARTIFACTS_DIR=\$HOME; bash build.sh 1> ~/build.out 2> ~/build.err &"
  echo "Sleep for 10 minutes for setting up VM (GPU) for test !"
  sleep 120s
  echo "Update commit id"
  commit_id=$(git rev-parse HEAD)
  echo $(commit_id) > commit.txt
  gsutil cp commit.txt gs://gcsfuse-ml-data/ci_artifacts/tf/resnet/
  if [ $(get_run_status) != "RUNNING" ];
  then
    echo "The model has not started"
    exit_status=1
  fi
elif [ $(get_run_status) == "RUNNING" ];
then
  # check if model has timed out
  start_time=$(gsutil cat gs://gcsfuse-ml-data/ci_artifacts/tf/resnet/start_time.txt)
  current_time=$(date +"%s")
  time_elapsed=$(expr $current_time - $start_time)
  if (( $(echo "$time_elapsed > $TIMEOUT" |bc -l) ));
  then
    echo "The tests have time out, start_time was $start_time, current time is $current_time"
    exit_status=1
  fi
elif [ $(get_run_status) == "ERROR" ];
then
  commit_id=$(get_commit_id)
  exit_status=1
elif [ $(get_run_status) == "COMPLETE" ];
then
  commit_id=$(get_commit_id)
  exit_status=0
  # change status back to start
  echo "START" > status.txt
  gsutil cp status.txt gs://gcsfuse-ml-data/ci_artifacts/tf/resnet/
else
  echo "Unknown state in status file. Please check."
  exit 1

fi

commit_id=$(get_commit_id)
copy_artifacts_to_gcs "tf-resnet-7d" $commit_id

echo "Below is the stdout of build on VM (GPU)"
gsutil cat gs://gcsfuse-ml-data/ci_artifacts/tf/resnet/$commit_id/build.out

echo "Below is the stderr of build on VM (GPU)"
gsutil cat gs://gcsfuse-ml-data/ci_artifacts/tf/resnet/$commit_id/build.err

exit $exit_status