#i!/bin/bash
set -e
sudo apt-get update
echo Installing fio
sudo apt-get install fio -y
echo Installing gcsfuse
curl -L -O https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v0.41.1/gcsfuse_0.41.1_amd64.deb
sudo dpkg --install gcsfuse_0.41.1_amd64.deb
cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"
echo Mounting gcs bucket
mkdir gcs
gcsfuse --implicit-dirs --max-conns-per-host 100 --disable-http2 gcs-fuse-dashboard-fio gcs 
chmod +x build.sh
./build.sh

