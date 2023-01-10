#!/bin/bash

# Installing docker engine
echo "Installing linux utility packages, like, lsb-release, curl..."
sudo apt-get update
sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

echo "Installing docker framework..."
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin -y

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/"

# Using the script from dlc_testing branch
mkdir dlc_setup
cd dlc_setup

mkdir container_artifacts && mkdir container_artifacts/logs && mkdir container_artifacts/output

# Building the dlc based dockerfile
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
cd gcsfuse
git checkout dlc_testing
sudo docker build . -f perfmetrics/dlc/Dockerfile -t dlc-gcsfuse --build-arg DLC_IMAGE_NAME=tf-gpu.2-10

# Running the container image
sudo docker run --runtime=nvidia --name tf_model_container --privileged -d \
-v ../container_artifacts/logs:/home/logs:rw,rshared \
-v ../container_artifacts/output:/home/output:rw,rshared --shm-size=24g dlc-gcsfuse:latest

# Mounting the GCS bucket with data for training
sudo docker exec tf_model_container bash -c 'BUCKET_NAME=ml-models-data-gcsfuse /dlc_testing/gcsfuse_mount.sh'

# Start training resnet model
sudo docker exec tf_model_container bash -c 'BUCKET_NAME=ml-models-data-gcsfuse /dlc_testing/gcsfuse_mount.sh'

# TODO cron setup

sudo docker logs -f tf_model_container

# TODO: copy logs to a bucket
