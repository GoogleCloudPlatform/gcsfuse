#i!/bin/bash
set -e
sudo apt-get update

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"

echo "Setting up a machine"
chmod +x ml_tests/setup.sh
source ml_tests/setup.sh