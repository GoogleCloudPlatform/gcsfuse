#! /bin/bash
# Copyright 2023 Google LLC
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
  # cat metadata.txt
  # ---
  #   value
  x=$(sed '2!d' metadata.txt) # fetch 2nd line -> " value"
  value=$(echo "$x" | sed 's/[[:space:]]//g') # remove preceeding spaces -> "value"
  rm metadata.txt
  echo $value
}

architecture=$(dpkg --print-architecture)
# Fetch metadata value of the key "RELEASE_VERSION"
RELEASE_VERSION=$(fetch_meta_data_value "RELEASE_VERSION")
echo RELEASE_VERSION="$RELEASE_VERSION"

# '~' is not accepted as docker build tag and git tag. Hence, we will use `_` instead of RELEASE_VERSION.
RELEASE_VERSION_TAG=$(echo $RELEASE_VERSION | tr '~' '_')
echo RELEASE_VERSION_TAG="$RELEASE_VERSION_TAG"

# Fetch metadata value of the key "UPLOAD_BUCKET"
UPLOAD_BUCKET=$(fetch_meta_data_value "UPLOAD_BUCKET")
echo UPLOAD_BUCKET="$UPLOAD_BUCKET"

# Fetch metadata value of the key "COMMIT_HASH"
COMMIT_HASH=$(fetch_meta_data_value "COMMIT_HASH")
echo COMMIT_HASH="$COMMIT_HASH"

sudo apt-get update
echo Install docker
sudo apt install apt-transport-https ca-certificates curl software-properties-common -y
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository "deb [arch=${architecture}] https://download.docker.com/linux/ubuntu focal stable"
apt-cache policy docker-ce
sudo apt install docker-ce -y
echo Install git
sudo apt-get install git -y
# It is require for multi-arch support
sudo apt-get install qemu-user-static binfmt-support
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
cd gcsfuse/tools/package_gcsfuse_docker/
git checkout "v$RELEASE_VERSION_TAG"
echo "Building docker for ${architecture} ..."
sudo docker buildx build --load . -t gcsfuse-release-${architecture}:"$RELEASE_VERSION_TAG" --build-arg GCSFUSE_VERSION="$RELEASE_VERSION" --build-arg ARCHITECTURE=${architecture} --build-arg BRANCH_NAME="v$RELEASE_VERSION_TAG" --build-arg COMMIT_HASH="$COMMIT_HASH" &> docker_${architecture}.log
gsutil cp docker_${architecture}.log gs://"$UPLOAD_BUCKET"/v"$RELEASE_VERSION"/
sudo docker run  -v $HOME/gcsfuse/release:/release gcsfuse-release-${architecture}:"$RELEASE_VERSION_TAG" cp -r /packages/. /release/v"$RELEASE_VERSION"

echo "Upload files in the bucket ..."
gsutil cp -r $HOME/gcsfuse/release/v"$RELEASE_VERSION" gs://"$UPLOAD_BUCKET"/
