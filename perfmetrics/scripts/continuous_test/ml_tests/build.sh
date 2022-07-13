#i!/bin/bash
set -e
sudo apt-get update
sudo add-apt-repository ppa:longsleep/golang-backports
sudo apt-get update
sudo apt-get install golang-go -y
sudo apt-get install golang-1.8-go -y

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"

echo "Setting up a machine"
chmod +x ml_tests/setup.sh
source ml_tests/setup.sh