#! /bin/bash
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

# The script is taking around 20-25 minutes on n2-standard-32 machine.
gcloud compute instances describe create-gcsfuse-binaries --zone us-west1-b --flatten="metadata[RELEASE_VERSION]" >>  release_version.txt
# e.g.
# cat release_version.txt
# ---
#   v1.0.7
x=$(sed '2!d' release_version.txt)
# echo "$x"
#   v1.0.7(contains preceding spaces)
# Remove spaces
RELEASE_VERSION=$(echo "$x" | sed 's/[[:space:]]//g')
# release version
# v1.0.7
rm release_version.txt
echo "$RELEASE_VERSION"
gcloud compute instances describe create-gcsfuse-binaries --zone us-west1-b --flatten="metadata[UPLOAD_BUCKET]" >>  bucket_name.txt
# e.g.
# cat bucket_name.txt
# ---
#   bucket name
x=$(sed '2!d' bucket_name.txt)
# echo "$x"
#   bucket_name(contains preceding spaces)
# Remove spaces
UPLOAD_BUCKET=$(echo "$x" | sed 's/[[:space:]]//g')
echo BUCKET NAME="$UPLOAD_BUCKET"
rm bucket_name.txt
sudo apt-get update
echo Install docker
sudo apt install apt-transport-https ca-certificates curl software-properties-common -y
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"
apt-cache policy docker-ce
sudo apt install docker-ce -y
echo Install git
sudo apt-get install git -y
sudo apt-get install qemu-user-static binfmt-support
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
cd gcsfuse
cd tools/package_gcsfuse_docker/
echo "Building docker for amd64"
sudo docker buildx build --load . -t gcsfuse-release-amd64:"$RELEASE_VERSION" --build-arg GCSFUSE_VERSION="$RELEASE_VERSION" --build-arg ARCHITECTURE=amd64 --platform=linux/amd64 &
echo "Building docker for arm64"
sudo docker buildx create --name mybuilder --bootstrap --use
sudo docker buildx build --load . -t gcsfuse-release-arm64:"$RELEASE_VERSION" --build-arg GCSFUSE_VERSION="$RELEASE_VERSION" --build-arg ARCHITECTURE=arm64 --platform=linux/arm64 &
echo "Waiting for both build to done"
wait
sudo docker run  -v $HOME/gcsfuse/release:/release gcsfuse-release-amd64:"$RELEASE_VERSION" cp -r /packages/. /release/v"$RELEASE_VERSION"
sudo docker run  -v $HOME/gcsfuse/release:/release gcsfuse-release-arm64:"$RELEASE_VERSION" cp -r /packages/. /release/v"$RELEASE_VERSION"
echo "Upload files in the bucket"
gsutil cp -r $HOME/gcsfuse/release/v"$RELEASE_VERSION" gs://"$UPLOAD_BUCKET"/
