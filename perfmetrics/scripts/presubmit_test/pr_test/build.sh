#!/bin/bash
set -e
sudo apt-get update
echo Installing git
sudo apt-get install git
echo Installing go-lang
sudo snap install go --classic
echo Installing fio
sudo apt-get install fio -y
mkdir master
cd master
echo cloning master branch code
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
cd /gcsfuse/perfmetrics/scripts
echo Mounting gcs bucket
mkdir -p gcs
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --experimental-enable-storage-client-library=true --disable-http2 --stackdriver-export-interval=30s"
BUCKET_NAME=gcs-fuse-loadtest
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
go run ../../. $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
chmod +x run_load_test_on_presubmit.sh
./run_load_test_on_presubmit.sh
cd
mkdir PR
cd PR
echo cloning PR branch code
git clone $GITHUB_PULL_REQUEST_URL
cd gcsfuse/perfmetrics/scripts
echo Mounting gcs bucket
mkdir -p gcs
GCSFUSE_FLAGS="--implicit-dirs --experimental-enable-storage-client-library=true --max-conns-per-host 100 --disable-http2 --stackdriver-export-interval=30s"
BUCKET_NAME=gcs-fuse-loadtest
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
go run ../../. $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
chmod +x run_load_test_on_presubmit.sh
./run_load_test_on_presubmit.sh