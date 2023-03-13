#!/bin/bash

# This will stop execution when any command will have non-zero status.
set -e

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"

echo "Setting up the machine with Docker and Nvidia Driver"
chmod +x ml_tests/setup_host.sh
source ml_tests/setup_host.sh

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
echo "Building docker image containing all pytorch libraries..."
sudo docker build . -f perfmetrics/scripts/ml_tests/pytorch/dino/Dockerfile --tag pytorch-gcsfuse

mkdir container_artifacts

echo "Running the docker image build in the previous step..."
sudo docker run --runtime=nvidia --name=pytorch_automation_container --privileged -d -v ${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/container_artifacts:/pytorch_dino/run_artifacts:rw,rshared \
--shm-size=128g pytorch-gcsfuse:latest

echo "Creating logrotate configuration..."
cat << EOF | tee ${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/gcsfuse_logrotate.conf
${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/container_artifacts/gcsfuse.log {
  rotate 10
  size 5G
  missingok
  notifempty
  compress
  dateext
  dateformat -%Y%m%d-%s
  copytruncate
}
EOF

# Make sure logrotate installed on the system.
if test -x /usr/sbin/logrotate ; then
  echo "Logrotate already installed on the system."
else
  echo "Installing logrotate on the system..."
  sudo apt-get install logrotate
fi

echo "Setting up cron job to rotate the gcsfuse_logs."
echo "0 */1 * * * /usr/sbin/logrotate ${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/gcsfuse_logrotate.conf --state ${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/gcsfuse_logrotate_status" | crontab -

# Wait for the script completion as well as logs output.
sudo docker logs -f pytorch_automation_container
