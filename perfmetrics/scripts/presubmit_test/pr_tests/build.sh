#!/bin/bash
set -e
sudo apt-get update
echo Installing git
sudo apt-get install git
echo Installing
sudo snap install go --classic
echo Installing fio
sudo apt-get install fio -y
echo cloning master branch code
git clone https://github.com/GoogleCloudPlatform/gcsfuse.git
cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"
echo Mounting gcs bucket
mkdir -p gcs
LOG_FILE=log-$(date '+%Y-%m-%d').txt
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --disable-http2 --debug_fuse --debug_gcs --log-file $LOG_FILE --log-format \"text\" --stackdriver-export-interval=30s"
BUCKET_NAME=gcs-fuse-dashboard-fio
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
go run ../../. $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
chmod +x run_load_test_and_fetch_metrics_pr.sh
./run_load_test_and_fetch_metrics_pr.sh
# Copying gcsfuse logs to bucket
gsutil -m cp $LOG_FILE gs://gcs-fuse-dashboard-fio/fio-gcsfuse-logs/

# Deleting logs older than 10 days
python3 utils/metrics_util.py gcs/fio-gcsfuse-logs/ 10