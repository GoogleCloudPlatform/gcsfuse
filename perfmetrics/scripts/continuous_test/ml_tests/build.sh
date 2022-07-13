#i!/bin/bash
set -e
sudo apt-get update
sudo add-apt-repository ppa:longsleep/golang-backports
sudo apt-get update
sudo apt-get install golang-go -y

echo 'export GOROOT=/usr/lib/go' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOROOT/bin:$GOPATH/bin' >> ~/.bashrc 
source ~/.bashrc

go version

cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse/perfmetrics/scripts"

echo "Setting up a machine"
chmod +x ml_tests/setup.sh
source ml_tests/setup.sh