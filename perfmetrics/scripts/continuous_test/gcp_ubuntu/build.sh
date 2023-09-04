#!/bin/bash
set -e
sudo apt-get update

# e.g. architecture=arm64 or amd64
architecture=$(dpkg --print-architecture)
echo "Installing git"
sudo apt-get install git
echo "Installing pip"
sudo apt-get install pip -y
echo "Installing go-lang 1.21.0"
wget -O go_tar.tar.gz https://go.dev/dl/go1.21.0.linux-${architecture}.tar.gz -q
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin
echo "Installing fio"
sudo apt-get install fio -y

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"

# Checkout back to master branch to use latest CI test scripts in master.
git checkout master

echo "Building and installing gcsfuse"
chmod +x perfmetrics/scripts/build_and_install_packge.sh
./perfmetrics/scripts/build_and_install_packge.sh

chmod +x perfmetrics/scripts/build_and_install_packge.sh
./perfmetrics/scripts/build_and_install_packge.sh

# Mounting gcs bucket
cd "./perfmetrics/scripts/"
echo "Mounting gcs bucket"
mkdir -p gcs
LOG_FILE=${KOKORO_ARTIFACTS_DIR}/gcsfuse-logs.txt
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --debug_fuse --debug_gcs --log-file $LOG_FILE --log-format \"text\" --stackdriver-export-interval=30s"
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
