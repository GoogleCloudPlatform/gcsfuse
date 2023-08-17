#!/bin/bash
# Running test only for when PR contains execute-perf-test or execute-integration-tests label
readonly EXECUTE_PERF_TEST_LABEL="execute-perf-test"
readonly EXECUTE_INTEGRATION_TEST_LABEL="execute-integration-tests"
readonly INTEGRATION_TEST_EXECUTION_TIME=24m

curl https://api.github.com/repos/GoogleCloudPlatform/gcsfuse/pulls/$KOKORO_GITHUB_PULL_REQUEST_NUMBER >> pr.json
perfTest=$(cat pr.json | grep $EXECUTE_PERF_TEST_LABEL)
integrationTests=$(cat pr.json | grep $EXECUTE_INTEGRATION_TEST_LABEL)
rm pr.json
perfTestStr="$perfTest"
integrationTestsStr="$integrationTests"
if [[ "$perfTestStr" != *$EXECUTE_PERF_TEST_LABEL*  &&  "$integrationTestsStr" != *$EXECUTE_INTEGRATION_TEST_LABEL* ]]
then
  echo "No need to execute tests"
  exit 0
fi

set -e
# It will take approx 80 minutes to run the script.
sudo apt-get update
echo Installing git
sudo apt-get install git
echo Installing go-lang  1.20.5
wget -O go_tar.tar.gz https://go.dev/dl/go1.20.5.linux-amd64.tar.gz -q
sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin
export CGO_ENABLED=0
cd "${KOKORO_ARTIFACTS_DIR}/github/gcsfuse"
# Fetch PR branch
echo '[remote "origin"]
         fetch = +refs/pull/*/head:refs/remotes/origin/pr/*' >> .git/config
git fetch origin -q

function execute_perf_test() {
  mkdir -p gcs
  GCSFUSE_FLAGS="--implicit-dirs --max-conns-per-host 100"
  BUCKET_NAME=presubmit-perf-tests
  MOUNT_POINT=gcs
  # The VM will itself exit if the gcsfuse mount fails.
  go run . $GCSFUSE_FLAGS $BUCKET_NAME $MOUNT_POINT
  # Running FIO test
  chmod +x perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh
  ./perfmetrics/scripts/presubmit/run_load_test_on_presubmit.sh
  sudo umount gcs
}

# execute perf tests.
if [[ "$perfTestStr" == *$EXECUTE_PERF_TEST_LABEL* ]];
then
 # Installing requirements
 echo Installing python3-pip
 sudo apt-get -y install python3-pip
 echo Installing libraries to run python script
 pip install google-cloud
 pip install google-cloud-vision
 pip install google-api-python-client
 pip install prettytable
 echo Installing fio
 sudo apt-get install fio -y

 # Executing perf tests for master branch
 git stash
 git checkout master
 # Store results
 touch result.txt
 echo Mounting gcs bucket for master branch and execute tests
 execute_perf_test


 # Executing perf tests for PR branch
 echo checkout PR branch
 git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER
 echo Mounting gcs bucket from pr branch and execute tests
 execute_perf_test

 # Show results
 echo showing results...
 python3 ./perfmetrics/scripts/presubmit/print_results.py
fi

# Execute integration tests.
if [[ "$integrationTestsStr" == *$EXECUTE_INTEGRATION_TEST_LABEL* ]];
then
  echo checkout PR branch
  git checkout pr/$KOKORO_GITHUB_PULL_REQUEST_NUMBER

  # Create bucket for integration tests.
  # The prefix for the random string
  bucketPrefix="gcsfuse-integration-test-"
  # The length of the random string
  length=5
  # Generate the random string
  random_string=$(tr -dc 'a-z0-9' < /dev/urandom | head -c $length)
  BUCKET_NAME=$bucketPrefix$random_string
  echo bucket name
  echo $BUCKET_NAME
  gcloud alpha storage buckets create gs://$BUCKET_NAME --project=gcs-fuse-test-ml --location=us-west1 --uniform-bucket-level-access

  # Executing integration tests
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/... -p 1 --integrationTest -v --testbucket=$BUCKET_NAME -timeout $INTEGRATION_TEST_EXECUTION_TIME

  # Delete bucket after testing.
  gcloud alpha storage rm --recursive gs://$BUCKET_NAME/
fi
