#!/bin/bash
set -e
sudo apt-get update

echo "Installing git"
sudo apt-get install git
echo "Installing pip"
sudo apt-get install pip -y
echo "Installing go-lang 1.20.4"
wget -O go_tar.tar.gz https://go.dev/dl/go1.20.4.linux-amd64.tar.gz
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
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
git checkout $commitId

echo "Executing integration tests"
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/... -p 1 --integrationTest -v --testbucket=gcsfuse-integration-test

# Checkout back to master branch to use latest CI test scripts in master.
git checkout master

echo "Building and installing gcsfuse"
# Build the gcsfuse package using the same commands used during release.
GCSFUSE_VERSION=0.0.0
sudo docker build ./tools/package_gcsfuse_docker/ -t gcsfuse:$commitId --build-arg GCSFUSE_VERSION=$GCSFUSE_VERSION --build-arg BRANCH_NAME=$commitId
sudo docker run -v $HOME/release:/release gcsfuse:$commitId cp -r /packages /release/
sudo dpkg -i $HOME/release/packages/gcsfuse_${GCSFUSE_VERSION}_amd64.deb

# Mounting gcs bucket
cd "./perfmetrics/scripts/"
echo "Mounting gcs bucket"
mkdir -p gcs
LOG_FILE=${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs.txt
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --enable-storage-client-library --debug_fuse --debug_gcs --log-file $LOG_FILE --log-format \"text\" --stackdriver-export-interval=30s"
BUCKET_NAME=periodic-perf-tests
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
gcsfuse $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT

# Executing perf tests
chmod +x run_load_test_and_fetch_metrics.sh
./run_load_test_and_fetch_metrics.sh

sudo umount $MOUNT_POINT

# ls_metrics test. This test does gcsfuse mount first and then do the testing.
cd "./ls_metrics"
chmod +x run_ls_benchmark.sh
./run_ls_benchmark.sh
