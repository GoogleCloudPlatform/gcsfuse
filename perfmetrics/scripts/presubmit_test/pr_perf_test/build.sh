#!/bin/bash
# It will take approx 80 minutes to run the script.
set -e
sudo apt-get update

echo Installing git
sudo apt-get install git

# Running test only for when title includes PerfTest
curl https://api.github.com/repos/GoogleCloudPlatform/gcsfuse/pulls/$KOKORO_GITHUB_PULL_REQUEST_NUMBER >> pr.json
title=$(cat pr.json | grep "title")
rm pr.json
perfTest=$( $title | grep "PerfTest")
if [ $perfTest == $title ]
then
  echo Installing python3-pip
  sudo apt-get -y install python3-pip
  echo Installing libraries to run python script
  pip install google-cloud
  pip install google-cloud-vision
  pip install google-api-python-client
  echo Installing go-lang 1.19.5
  wget -O go_tar.tar.gz https://go.dev/dl/go1.19.5.linux-amd64.tar.gz
  sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
  export PATH=$PATH:/usr/local/go/bin
  echo Installing fio
  sudo apt-get install fio -y

  cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
  echo Mounting gcs bucket for master branch
  mkdir -p gcs
  GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100 --experimental-enable-storage-client-library=true"
  BUCKET_NAME=presubmit-perf-test
  MOUNT_POINT=gcs
  # The VM will itself exit if the gcsfuse mount fails.
  go run . $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
  touch result.txt
  echo "Results of the master branch" >> result.txt
  # Running FIO test
  chmod +x perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh
  ./perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh
  sudo umount gcs
  # Fetch PR branch
  echo '[remote "origin"]
          fetch = +refs/pull/*/head:refs/remotes/origin/pr/*' >> .git/config
  git fetch origin
  echo checkout PR branch
  git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER
  echo Mounting gcs bucket from pr branch
  mkdir -p gcs
  # The VM will itself exit if the gcsfuse mount fails.
  go run . $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
  echo "Results of the PR branch" >> result.txt
  # Running FIO test
  chmod +x perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh
  ./perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh

  echo showing results...
  cat result.txt
else
  echo "No need to execute tests"