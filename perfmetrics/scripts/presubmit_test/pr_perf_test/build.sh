#!/bin/bash
set -e
sudo apt-get update

echo "Installing git"
sudo apt-get install git
echo "Installing go-lang 1.20.5"
wget -O go_tar.tar.gz https://go.dev/dl/go1.20.5.linux-$(dpkg --print-architecture).tar.gz -q
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin
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
git stash
git checkout $commitId

echo "Building and installing gcsfuse"
# Build the gcsfuse package using the same commands used during release.
GCSFUSE_VERSION=0.0.0
sudo docker build ./tools/package_gcsfuse_docker/ -t gcsfuse:$commitId --build-arg GCSFUSE_VERSION=$GCSFUSE_VERSION --build-arg BRANCH_NAME=$commitId --build-arg ARCHITECTURE=arm64
sudo docker run -v $HOME/release:/release gcsfuse:$commitId cp -r /packages /release/
ls $HOME/release/packages/
sudo dpkg -i $HOME/release/packages/gcsfuse_${GCSFUSE_VERSION}_arm64.deb

echo "Executing integration tests"
GODEBUG=asyncpreemptoff=1 CGO_ENABLED=0 go test ./tools/integration_tests/... -p 1 --testInstalledPackage --integrationTest -v --testbucket=integration-test-tulsishah-2 -timeout 24m
