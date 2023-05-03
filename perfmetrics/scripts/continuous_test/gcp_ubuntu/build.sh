#!/bin/bash
set -e
sudo apt-get update

echo Installing git
sudo apt-get install git
echo Installing pip
sudo apt-get install pip -y
echo Installing go-lang 1.20.3
wget -O go_tar.tar.gz https://go.dev/dl/go1.20.3.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin
echo Installing fio
sudo apt-get install fio -y

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
# Get the latest commitId of yesterday in the log file
commitId=$(git log --before='yesterday 23:59:59' --max-count=1 --pretty=%H)
git checkout $commitId

echo Building and installing gcsfuse
go install ./tools/build_gcsfuse
mkdir $HOME/temp
$HOME/go/bin/build_gcsfuse ./ $HOME/temp/ $commitId
sudo cp ~/temp/bin/gcsfuse /usr/bin
sudo cp ~/temp/sbin/mount.gcsfuse /sbin

# Executing integration tests
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/... -p 1 --integrationTest -v --testbucket=gcsfuse-integration-tests

# Mounting gcs bucket
cd "./perfmetrics/scripts/"
echo Mounting gcs bucket
mkdir -p gcs
LOG_FILE=log-$(date '+%Y-%m-%d').txt
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --enable-storage-client-library --debug_fuse --debug_gcs --log-file $LOG_FILE --log-format \"text\" --stackdriver-export-interval=30s"
BUCKET_NAME=gcs-fuse-dashboard-fio
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
gcsfuse $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT

# Executing perf tests
chmod +x run_load_test_and_fetch_metrics.sh
./run_load_test_and_fetch_metrics.sh
# Copying gcsfuse logs to bucket
gsutil -m cp $LOG_FILE gs://gcs-fuse-dashboard-fio/fio-gcsfuse-logs/

# Deleting logs older than 10 days
python3 utils/metrics_util.py gcs/fio-gcsfuse-logs/ 10

# ls_metrics test
cd "./ls_metrics"
chmod +x run_ls_benchmark.sh
./run_ls_benchmark.sh
