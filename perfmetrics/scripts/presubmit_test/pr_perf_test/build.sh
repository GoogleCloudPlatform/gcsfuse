#!/bin/bash
set -e
sudo apt-get update

echo "Installing git"
sudo apt-get install git
echo "Installing pip"
sudo apt-get install pip -y
echo "Installing go-lang 1.20.3"
wget -O go_tar.tar.gz https://go.dev/dl/go1.20.3.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin
echo "Installing fio"
sudo apt-get install fio -y
echo "Installing docker "
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin -y

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
# Get the latest commitId of yesterday in the log file. Build gcsfuse and run
# integration tests using code upto that commit.
echo '[remote "origin"]
         fetch = +refs/pull/*/head:refs/remotes/origin/pr/*' >> .git/config
git fetch origin
echo checkout PR branch
git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER

echo "Building and installing gcsfuse"
# Build the gcsfuse package using the same commands used during release.

# Mounting gcs bucket

echo "Mounting gcs bucket"
mkdir -p gcs
LOG_FILE=${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs.txt
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --enable-storage-client-library --debug_fuse --debug_gcs --log-file $LOG_FILE --log-format \"text\" --stackdriver-export-interval=30s"
BUCKET_NAME=periodic-perf-tests
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
go run . $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT

cd "./perfmetrics/scripts/"
# Executing perf tests
chmod +x run_load_test_and_fetch_metrics.sh
./run_load_test_and_fetch_metrics.sh

sudo umount $MOUNT_POINT