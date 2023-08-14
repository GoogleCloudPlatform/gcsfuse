#!/bin/bash
# Running test only for when PR contains execute-perf-test label
curl https://api.github.com/repos/GoogleCloudPlatform/gcsfuse/pulls/$KOKORO_GITHUB_PULL_REQUEST_NUMBER >> pr.json
perfTest=$(cat pr.json | grep "execute-perf-test")
rm pr.json
perfTestStr="$perfTest"
if [[ "$perfTestStr" != *"execute-perf-test"* ]]
then
  echo "No need to execute tests"
  exit 0
fi

# It will take approx 80 minutes to run the script.
set -e
sudo apt-get update
echo Installing git
sudo apt-get install git
echo Installing python3-pip
sudo apt-get -y install python3-pip
echo Installing libraries to run python script
pip install google-cloud
pip install google-cloud-vision
pip install google-api-python-client
pip install prettytable
echo Installing go-lang  1.20.5
wget -O go_tar.tar.gz https://go.dev/dl/go1.20.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin
echo Installing fio
sudo apt-get install fio -y

# Run on master branch
cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
# Executing integration tests
GODEBUG=asyncpreemptoff=1 CGO_ENABLED=0 go test ./tools/integration_tests/... -p 1 --integrationTest -v --testbucket=gcsfuse-integration-test -timeout 15m
