#!/bin/bash
# It will take approx 1.5-2 hours to run the script.
set -e
sudo apt-get update
echo Installing git
sudo apt-get install git

echo Installing python3-pip
sudo apt-get -y install python3-pip

echo Installing libraries to run python script
pip install google-cloud
pip install google-cloud-vision
pip install google-api-python-client

echo Installing go-lang 1.19.5
wget -O go_tar.tar.gz https://go.dev/dl/go1.19.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin

echo Installing fio
sudo apt-get install fio -y

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"

echo Mounting gcs bucket for master branch
mkdir -p gcs
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --experimental-enable-storage-client-library=true --disable-http2"
BUCKET_NAME=gcs-fuse-loadtest
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
go run . $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
# Running FIO test
chmod +x perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh
./perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh
sudo umount perfmetrics/scripts/gcs
# Fetch PR branch
echo '[remote "origin"]
        fetch = +refs/pull/*/head:refs/remotes/origin/pr/*' >> .git/config
git fetch origin

echo checkout PR branch
git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER

echo Mounting gcs bucket from pr branch
mkdir -p gcs
# The VM will itself exit if the gcsfuse mount fails.
go run . $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
chmod +x perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh
./perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh