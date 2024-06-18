#! /bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
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

# This script will create gcsfuse package for both amd64 and arm64 machines.
# After creation of package we are uploading it on gcs bucket.

# The script is taking around 20-25 minutes on n2-standard-32 machine.

# The script will create following packages and upload it on the bucket.
# gcsfuse_RELEASE_VERSION_amd64.deb
# gcsfuse_RELEASE_VERSION_arm64.deb
# gcsfuse-RELEASE_VERSION-1.aarch64.rpm
# gcsfuse-RELEASE_VERSION-1.x86_64.rpm

set -e
# create-gcsfuse-packages VM is a fixed VM in GCP project gcs-fuse-test

# Function to fetch metadata value of the key.
function fetch_meta_data_value() {
  metadata_key=$1
  # Fetch metadata value of the key
  gcloud compute instances describe create-gcsfuse-packages --zone us-west1-b --flatten="metadata[$metadata_key]" >>  metadata.txt
  # cat metadata.txt.txt
  # ---
  #   value
  x=$(sed '2!d' metadata.txt)
  #   value(contains preceding spaces)
  # Remove spaces
  # value
  value=$(echo "$x" | sed 's/[[:space:]]//g')
  # echo $value
  # value
  rm metadata.txt
  echo $value
}

# Function to check docker build failure.
function check_docker_build_failure() {
    if [[ $? -ne 0 ]]; then
      echo "docker build fails."
      exit_code=1
    fi
}

function exit_in_failure() {
    if [[ $exit_code == 1 ]]; then
       exit 1
    fi
}

# Fetch metadata value of the key "RELEASE_VERSION"
RELEASE_VERSION=$(fetch_meta_data_value "RELEASE_VERSION")
echo RELEASE_VERSION="$RELEASE_VERSION"

# '~' is not accepted as docker build tag and git tag. Hence, we will use `_` instead of RELEASE_VERSION.
RELEASE_VERSION_TAG=$(echo $RELEASE_VERSION | tr '~' '_')
echo RELEASE_VERSION_TAG="$RELEASE_VERSION_TAG"

# Fetch metadata value of the key "UPLOAD_BUCKET"
UPLOAD_BUCKET=$(fetch_meta_data_value "UPLOAD_BUCKET")
echo UPLOAD_BUCKET="$UPLOAD_BUCKET"

sudo apt-get update
echo Install docker
sudo apt install apt-transport-https ca-certificates curl software-properties-common -y
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"
apt-cache policy docker-ce
sudo apt install docker-ce -y
echo Install git
sudo apt-get install git -y
# It is require for multi-arch support
sudo apt-get install qemu-user-static binfmt-support
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
cd gcsfuse/tools/package_gcsfuse_docker/
# Setting set +e to capture error output in log file and send it on the bucket.
set +e
exit_code=0
echo "Building docker for amd64 ..."
sudo docker buildx build --load . -t gcsfuse-release-amd64:"$RELEASE_VERSION_TAG" --build-arg GCSFUSE_VERSION="$RELEASE_VERSION" --build-arg ARCHITECTURE=amd64 --build-arg BRANCH_NAME="v$RELEASE_VERSION_TAG" --platform=linux/amd64 &> docker_amd64.log &
pid1=$!
echo "Building docker for arm64 ..."
# It is necessary for cross-platform image building because it creates a builder instance that is capable of
# building images for multiple architectures.
sudo docker buildx create --name mybuilder --bootstrap --use
sudo docker buildx build --load . -t gcsfuse-release-arm64:"$RELEASE_VERSION_TAG" --build-arg GCSFUSE_VERSION="$RELEASE_VERSION" --build-arg ARCHITECTURE=arm64 --build-arg BRANCH_NAME="v$RELEASE_VERSION_TAG" --platform=linux/arm64 &> docker_arm64.log &
pid2=$!
echo "Waiting for both builds to complete ..."
wait $pid1
check_docker_build_failure
gsutil cp docker_amd64.log gs://"$UPLOAD_BUCKET"/v"$RELEASE_VERSION"/
wait $pid2
check_docker_build_failure
gsutil cp docker_arm64.log gs://"$UPLOAD_BUCKET"/v"$RELEASE_VERSION"/

# Exit if any of the build fails.
exit_in_failure

set -e
# Below steps are taking less than one second, so we are not parallelising them.
# Copy packages from docket container to disk.
sudo docker run  -v $HOME/gcsfuse/release:/release gcsfuse-release-amd64:"$RELEASE_VERSION_TAG" cp -r /packages/. /release/v"$RELEASE_VERSION"
sudo docker run  -v $HOME/gcsfuse/release:/release gcsfuse-release-arm64:"$RELEASE_VERSION_TAG" cp -r /packages/. /release/v"$RELEASE_VERSION"

echo "Upload files in the bucket ..."
gsutil cp -r $HOME/gcsfuse/release/v"$RELEASE_VERSION" gs://"$UPLOAD_BUCKET"/
