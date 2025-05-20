#!/bin/bash
# Copyright 2023 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This will stop execution when any command will have non-zero status.

# true or false to run e2e tests on installedPackage
RUN_E2E_TESTS_ON_PACKAGE=$1

# Pass "true" to skip few non-essential tests.
# By default, this script runs all the integration tests.
SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE=$2

# e.g. us-west1
BUCKET_LOCATION=$3

# Pass "true" to run e2e tests on TPC endpoint.
# The default value will be false.
RUN_TEST_ON_TPC_ENDPOINT=false
if [ $4 != "" ]; then
  RUN_TEST_ON_TPC_ENDPOINT=$4
fi
INTEGRATION_TEST_TIMEOUT_IN_MINS=90

RUN_TESTS_WITH_PRESUBMIT_FLAG=false
if [ $# -ge 5 ] ; then
  # This parameter is set to true by caller, only for presubmit runs.
  RUN_TESTS_WITH_PRESUBMIT_FLAG=$5
fi

# 6th parameter is set to enable/disable run for zonal bucket(s).
# If it is set to true, then the run will be only on zonal bucket(s),
# otherwise the run will only on non-zonal bucket(s).
RUN_TESTS_WITH_ZONAL_BUCKET=false
if [[ $# -ge 6 ]] ; then
  if [[ "$6" == "true" ]]; then
    RUN_TESTS_WITH_ZONAL_BUCKET=true
  elif [[ "$6" != "false" ]]; then
    echo "Error: Invalid value for 6th argument: "$6" . Expected: true or false."
    exit 1
  fi
fi

if ${RUN_TESTS_WITH_ZONAL_BUCKET}; then
  if [ "${BUCKET_LOCATION}" != "us-west4" ] && [ "${BUCKET_LOCATION}" != "us-central1" ]; then
    >&2 echo "For enabling zonal bucket run, BUCKET_LOCATION should be one of: us-west4, us-central1; passed: ${BUCKET_LOCATION}"
    exit 1
  fi
fi

if [ "$#" -lt 3 ]
then
  echo "Incorrect number of arguments passed, please refer to the script and pass the three arguments required..."
  exit 1
fi

if [ "$SKIP_NON_ESSENTIAL_TESTS_ON_PACKAGE" == true ]; then
  GO_TEST_SHORT_FLAG="-short"
  echo "Setting the flag to skip few un-important integration tests."
  INTEGRATION_TEST_TIMEOUT_IN_MINS=$((INTEGRATION_TEST_TIMEOUT_IN_MINS-20))
fi

# Pass flag "-presubmit" to 'go test' command and lower timeout for presubmit runs.
if [ "$RUN_TESTS_WITH_PRESUBMIT_FLAG" == true ]; then
  echo "This is a presubmit-run, which skips some tests."
  PRESUBMIT_RUN_FLAG="-presubmit"
  INTEGRATION_TEST_TIMEOUT_IN_MINS=$((INTEGRATION_TEST_TIMEOUT_IN_MINS-10))
fi

INTEGRATION_TEST_TIMEOUT=""${INTEGRATION_TEST_TIMEOUT_IN_MINS}"m"
echo "Setting the integration test timeout to: $INTEGRATION_TEST_TIMEOUT"

readonly RANDOM_STRING_LENGTH=5
# Test directory arrays
TEST_DIR_PARALLEL=(
  "monitoring"
  "local_file"
  "log_rotation"
  "mounting"
  "read_cache"
  # "grpc_validation"
  "gzip"
  "write_large_files"
  "list_large_dir"
  "rename_dir_limit"
  "read_large_files"
  "explicit_dir"
  "implicit_dir"
  "interrupt"
  "operations" # operations package is here
  "kernel_list_cache"
  "concurrent_operations"
  "benchmarking"
  "mount_timeout"
  "stale_handle"
  "negative_stat_cache"
  "streaming_writes"
)

# These tests never become parallel as it is changing bucket permissions.
TEST_DIR_NON_PARALLEL=(
  "readonly"
  "managed_folders"
  "readonly_creds"
)

# Subset of TEST_DIR_PARALLEL,
# but only those tests which currently
# pass for zonal buckets.
TEST_DIR_PARALLEL_FOR_ZB=(
  "benchmarking"
  "explicit_dir"
  "gzip"
  "implicit_dir"
  "interrupt"
  "kernel_list_cache"
  "local_file"
  "log_rotation"
  "monitoring"
  "mount_timeout"
  "mounting"
  "negative_stat_cache"
  "operations" # operations package is here
  "read_cache"
  "read_large_files"
  "rename_dir_limit"
  "stale_handle"
  "streaming_writes"
  "write_large_files"
  "unfinalized_object"
)

# Subset of TEST_DIR_NON_PARALLEL,
# but only those tests which currently
# pass for zonal buckets.
TEST_DIR_NON_PARALLEL_FOR_ZB=(
  "concurrent_operations"
  "list_large_dir"
  "managed_folders"
  "readonly"
  "readonly_creds"
)

# Create a temporary file to store the log file name.
TEST_LOGS_FILE=$(mktemp)

# Delete contents of the buckets (and then the buckets themselves) whose names are in the passed file.
# Args: <bucket-names-file>
function delete_buckets_listed_in_file() {
	local bucketNamesFile="${@}"
	if test -f "${bucketNamesFile}"; then
		cat "${bucketNamesFile}" | while read bucket; do
			# Only if bucket-name is non-empty and contains
			# something other than spaces.
			if [ -n "${bucket}" ] && [ -n "${bucket// }" ]; then
				# Delete the bucket and its contents.
				if ! gcloud -q storage rm -r --verbosity=none gs://${bucket} ; then
					>&2 echo "Failed to delete bucket ${bucket} !"
				fi
			fi
		done
		# At the end, delete the bucket-names file itself.
		rm "${bucketNamesFile}"
	else
		echo "file ${bucketNamesFile} not found !"
	fi
}

function upgrade_gcloud_version() {
  sudo apt-get update
  # Upgrade gcloud version.
  # Kokoro machine's outdated gcloud version prevents the use of the "managed-folders" feature.
  gcloud version
  wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
  sudo tar xzf gcloud.tar.gz && sudo cp -r google-cloud-sdk /usr/local && sudo rm -r google-cloud-sdk
  sudo /usr/local/google-cloud-sdk/install.sh
  export PATH=/usr/local/google-cloud-sdk/bin:$PATH
  echo 'export PATH=/usr/local/google-cloud-sdk/bin:$PATH' >> ~/.bashrc
  gcloud version && rm gcloud.tar.gz
  sudo /usr/local/google-cloud-sdk/bin/gcloud components update
  sudo /usr/local/google-cloud-sdk/bin/gcloud components install alpha
}

function install_packages() {
  # e.g. architecture=arm64 or amd64
  architecture=$(dpkg --print-architecture)
  echo "Installing go-lang 1.24.0..."
  wget -O go_tar.tar.gz https://go.dev/dl/go1.24.0.linux-${architecture}.tar.gz -q
  sudo rm -rf /usr/local/go && tar -xzf go_tar.tar.gz && sudo mv go /usr/local
  export PATH=$PATH:/usr/local/go/bin
  sudo apt-get install -y python3
  # install python3-setuptools tools.
  sudo apt-get install -y gcc python3-dev python3-setuptools
  # Downloading composite object requires integrity checking with CRC32c in gsutil.
  # it requires to install crcmod.
  sudo apt install -y python3-crcmod
  # Install gotestsum for XML output
  echo "Installing gotestsum..."
  go install gotest.tools/gotestsum@latest
  # Ensure gotestsum is in PATH (might require adjusting based on GOPATH/GOBIN)
  # If GOBIN is not in PATH, this might be needed:
  export PATH=$(go env GOPATH)/bin:$PATH
}

function create_bucket() {
  bucket_prefix=$1
  local -r project_id="gcs-fuse-test-ml"
  # Generate bucket name with random string
  bucket_name=${bucket_prefix}$(date +%Y%m%d-%H%M%S)"-"$(tr -dc 'a-z0-9' < /dev/urandom | head -c $RANDOM_STRING_LENGTH)
  # We are using gcloud alpha because gcloud storage is giving issues running on Kokoro
  gcloud alpha storage buckets create gs://$bucket_name --project=$project_id --location=$BUCKET_LOCATION --uniform-bucket-level-access
  echo $bucket_name
}

function create_hns_bucket() {
  local -r hns_project_id="gcs-fuse-test"
  # Generate bucket name with random string.
  # Adding prefix `golang-grpc-test` to white list the bucket for grpc
  # so that we can run grpc related e2e tests.
  bucket_name="golang-grpc-test-gcsfuse-e2e-tests-hns-"$(date +%Y%m%d-%H%M%S)"-"$(tr -dc 'a-z0-9' < /dev/urandom | head -c $RANDOM_STRING_LENGTH)
  gcloud alpha storage buckets create gs://$bucket_name --project=$hns_project_id --location=$BUCKET_LOCATION --uniform-bucket-level-access --enable-hierarchical-namespace
  echo "$bucket_name"
}

function create_zonal_bucket() {
  local -r project_id="gcs-fuse-test-ml"
  local -r region=${BUCKET_LOCATION}
  local -r zone=${region}"-a"

  local -r hns_project_id="gcs-fuse-test"
  # Generate bucket name with random string.
  bucket_name="gcsfuse-e2e-tests-zb-"$(date +%Y%m%d-%H%M%S)"-"$(tr -dc 'a-z0-9' < /dev/urandom | head -c $RANDOM_STRING_LENGTH)
  gcloud alpha storage buckets create gs://$bucket_name --project=$project_id --location=$region --placement=${zone} --default-storage-class=RAPID --uniform-bucket-level-access --enable-hierarchical-namespace
  echo "${bucket_name}"
}

function run_non_parallel_tests() {
  local exit_code=0
  local -n test_array=$1
  local bucket_name_non_parallel=$2
  local zonal=false
  if [ $# -ge 3 ] && [ "$3" = "true" ] ; then
    zonal=true
  fi

  for test_dir_np in "${test_array[@]}"
  do
    test_path_non_parallel="./tools/integration_tests/$test_dir_np"
    local log_file="/tmp/${test_dir_np}_${bucket_name_non_parallel}.log"
    echo $log_file >> $TEST_LOGS_FILE

    local junit_report_file="/tmp/junit_report_${test_dir_np}_${bucket_name_non_parallel}.xml"

    echo "Running test package in non-parallel (with zonal=${zonal}): ${test_dir_np} ..."
    if [[ "${test_dir_np}" == "operations" && -x "$(command -v gotestsum)" ]]; then
        echo "Using gotestsum for JUnit report: ${junit_report_file}"
        GODEBUG=asyncpreemptoff=1 gotestsum --junitfile "${junit_report_file}" -- $test_path_non_parallel -p 1 $GO_TEST_SHORT_FLAG $PRESUBMIT_RUN_FLAG --zonal=${zonal} --integrationTest -v --testbucket=$bucket_name_non_parallel --testInstalledPackage=$RUN_E2E_TESTS_ON_PACKAGE -timeout $INTEGRATION_TEST_TIMEOUT > "$log_file" 2>&1
    elif [[ "${test_dir_np}" == "operations" ]]; then
        echo "Warning: gotestsum not found. Skipping JUnit for ${test_dir_np}."
        GODEBUG=asyncpreemptoff=1 go test $test_path_non_parallel -p 1 $GO_TEST_SHORT_FLAG $PRESUBMIT_RUN_FLAG --zonal=${zonal} --integrationTest -v --testbucket=$bucket_name_non_parallel --testInstalledPackage=$RUN_E2E_TESTS_ON_PACKAGE -timeout $INTEGR