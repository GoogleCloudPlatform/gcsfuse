#i!/bin/bash
set -e
sudo apt-get update
echo Installing fio
sudo apt-get install fio -y
echo Installing gcsfuse
GCSFUSE_VERSION=0.41.4
curl -L -O https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v$GCSFUSE_VERSION/gcsfuse_"$GCSFUSE_VERSION"_amd64.deb
sudo dpkg --install gcsfuse_"$GCSFUSE_VERSION"_amd64.deb
cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"
echo Mounting gcs bucket
mkdir -p gcs
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --disable-http2 --experimental-stackdriver-export-interval=30s"
BUCKET_NAME=gcs-fuse-dashboard-fio
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
gcsfuse $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
chmod +x run_load_test_and_fetch_metrics.sh
./run_load_test_and_fetch_metrics.sh
