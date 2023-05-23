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
echo Installing go-lang  1.20.3
wget -O go_tar.tar.gz https://go.dev/dl/go1.20.3.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin
echo Installing fio
sudo apt-get install fio -y

# Run on master branch
cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
echo Mounting gcs bucket for master branch
mkdir -p gcs
LOG_FILE=logs-new.txt
GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --debug_fuse --debug_gcs --log-file $LOG_FILE"
BUCKET_NAME=presubmit-perf-tests
MOUNT_POINT=gcs
# The VM will itself exit if the gcsfuse mount fails.
go run . $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
touch result.txt
# Running FIO test


# Fetch PR branch
echo '[remote "origin"]
         fetch = +refs/pull/*/head:refs/remotes/origin/pr/*' >> .git/config
git fetch origin
echo checkout PR branch
git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER

echo new code....
chmod +x perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh
./perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh
sudo umount gcs

# Executing perf tests


echo showing results...