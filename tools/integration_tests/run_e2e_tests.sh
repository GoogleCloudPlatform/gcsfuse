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

# 7th parameter is to determine whether we want to disable build by the script
# and let every test package build its own GCSFuse binary.
BUILD_BINARY_IN_SCRIPT=true
if [[ $# -ge 7 ]] ; then
  if [[ "$7" == "false" ]]; then
    BUILD_BINARY_IN_SCRIPT=false
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
  "operations"
  "kernel_list_cache"
  "concurrent_operations"
  "benchmarking"
  "mount_timeout"
  "stale_handle"
  "negative_stat_cache"
  "streaming_writes"
  "inactive_stream_timeout"
  "cloud_profiler"
  "release_version"
  "readdirplus"
  "dentry_cache"
  "buffered_read"
  "requester_pays_bucket"
  "flag_optimizations"
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
  "operations"
  "rapid_appends"
  "read_cache"
  "read_large_files"
  "rename_dir_limit"
  "stale_handle"
  "streaming_writes"
  "write_large_files"
  "unfinalized_object"
  "release_version"
  "readdirplus"
  "dentry_cache"
  "flag_optimizations"
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

# This variable will store the path if the script builds GCSFuse binaries (gcsfuse, mount.gcsfuse)
BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR=""
# This variable will hold flag and its value to be passed to GCSFuse tests (--gcsfuse_prebuilt_dir=...)
USE_PREBUILT_GCSFUSE_BINARY=""

build_gcsfuse_once() {
  local build_output_dir # For the final gcsfuse binaries
  build_output_dir=$(mktemp -d -t gcsfuse_e2e_run_build_XXXXXX)
  echo "GCSFuse binaries will be built in ${build_output_dir}..."

  local gcsfuse_src_dir
  # Determine GCSFuse source directory
  # If this script is in tools/integration_tests, project root is ../../
  SCRIPT_DIR_REALPATH=$(realpath "$(dirname "${BASH_SOURCE[0]}")")
  gcsfuse_src_dir=$(realpath "${SCRIPT_DIR_REALPATH}/../../")

  if [[ ! -f "${gcsfuse_src_dir}/go.mod" ]]; then
    echo "Error: Could not reliably determine GCSFuse project root from ${SCRIPT_DIR_REALPATH}. Expected go.mod at ${gcsfuse_src_dir}" >&2
    rm -rf "${build_output_dir}"
    exit 1
  fi
  echo "Using GCSFuse source directory: ${gcsfuse_src_dir}"

  echo "Building GCSFuse using 'go run ./tools/build_gcsfuse/main.go'..."
  (cd "${gcsfuse_src_dir}" && go run ./tools/build_gcsfuse/main.go . "${build_output_dir}" "e2e-$(date +%s)")
  if [ $? -ne 0 ]; then
    echo "Error building GCSFuse binaries using 'go run ./tools/build_gcsfuse/main.go'."
    rm -rf "${build_output_dir}" # Clean up created temp dir
    return 1
  fi

  # Set the directory path for use by the script (to form the go test flag)
  BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR="${build_output_dir}"
  echo "GCSFuse binaries built by script in: ${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}"
  echo "GCSFuse executable: ${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}/bin/gcsfuse"
  return 0
}


cleanup_gcsfuse_once() {
  if [ -n "${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}" ] && [ -d "${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}" ]; then
    echo "Cleaning up GCSFuse build directory created by script: ${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}"
    rm -rf "${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}"
  fi
}

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
  # Install latest gcloud.
  ./perfmetrics/scripts/install_latest_gcloud.sh
  export PATH="/usr/local/google-cloud-sdk/bin:$PATH"
}

function install_packages() {
  # Install required go version.
  ./perfmetrics/scripts/install_go.sh "1.24.5"
  export PATH="/usr/local/go/bin:$PATH"
  
  sudo apt-get update
  sudo apt-get install -y python3
  # install python3-setuptools tools.
  sudo apt-get install -y gcc python3-dev python3-setuptools
  # Downloading composite object requires integrity checking with CRC32c in gsutil.
  # it requires to install crcmod.
  sudo apt install -y python3-crcmod
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
    # To make it clear whether tests are running on a flat or HNS bucket, We kept the log file naming
    # convention to include the bucket name as a suffix (e.g., package_name_bucket_name).
    local log_file="/tmp/${test_dir_np}_${bucket_name_non_parallel}.log"
    echo $log_file >> $TEST_LOGS_FILE

    # Executing integration tests
    echo "Running test package in non-parallel (with zonal=${zonal}): ${test_dir_np} ..."
    GODEBUG=asyncpreemptoff=1 go test $test_path_non_parallel -p 1 $GO_TEST_SHORT_FLAG $PRESUBMIT_RUN_FLAG --zonal=${zonal} --integrationTest -v --testbucket=$bucket_name_non_parallel --testInstalledPackage=$RUN_E2E_TESTS_ON_PACKAGE $USE_PREBUILT_GCSFUSE_BINARY -timeout $INTEGRATION_TEST_TIMEOUT > "$log_file" 2>&1
    exit_code_non_parallel=$?
    if [ $exit_code_non_parallel != 0 ]; then
      exit_code=$exit_code_non_parallel
      echo "test fail in non parallel on package (with zonal=${zonal}): " $test_dir_np
    else
      echo "Passed test package in non-parallel (with zonal=${zonal}): " $test_dir_np
    fi
  done
  return $exit_code
}

function run_parallel_tests() {
  local exit_code=0
  local -n test_array=$1
  local bucket_name_parallel=$2
  local zonal=false
  if [ $# -ge 3 ] && [ "$3" = "true" ] ; then
    zonal=true
  fi
  local benchmark_flags=""
  declare -A pids

  for test_dir_p in "${test_array[@]}"
  do
    # Unlike regular tests,benchmark tests are not executed by default when using go test .
    # The -bench flag yells go test to run the benchmark tests and report their results by
    # enabling the benchmarking framework.
    # The -benchtime flag specifies exact number of iterations a benchmark should run , in this
    # case, setting this to 100 to avoid flakiness. 
    if [ $test_dir_p == "benchmarking" ]; then
      benchmark_flags="-bench=. -benchtime=100x"
    fi
    test_path_parallel="./tools/integration_tests/$test_dir_p"
    # To make it clear whether tests are running on a flat or HNS bucket, We kept the log file naming
    # convention to include the bucket name as a suffix (e.g., package_name_bucket_name).
    local log_file="/tmp/${test_dir_p}_${bucket_name_parallel}.log"
    echo $log_file >> $TEST_LOGS_FILE
    # Executing integration tests
    echo "Queueing up test package in parallel (with zonal=${zonal}): ${test_dir_p} ..."
    GODEBUG=asyncpreemptoff=1 go test $test_path_parallel $GO_TEST_SHORT_FLAG $PRESUBMIT_RUN_FLAG --zonal=${zonal} $benchmark_flags -p 1 --integrationTest -v --testbucket=$bucket_name_parallel --testInstalledPackage=$RUN_E2E_TESTS_ON_PACKAGE $USE_PREBUILT_GCSFUSE_BINARY -timeout $INTEGRATION_TEST_TIMEOUT > "$log_file" 2>&1 &
    pid=$!  # Store the PID of the background process
    echo "Queued up test package in parallel (with zonal=${zonal}): ${test_dir_p} with pid=${pid}"
    pids[${test_dir_p}]=${pid} # Optionally add the PID to an array for later
  done

  # Wait for processes and collect exit codes
  for package_name in "${!pids[@]}"; do
    pid="${pids[${package_name}]}"
    echo "Waiting on test package ${package_name} (with zonal=${zonal}) through pid=${pid} ..."
    # What if the process for this test package completed long back and its PID got
    # re-assigned to another process since then ?
    wait $pid
    exit_code_parallel=$?
    if [ $exit_code_parallel != 0 ]; then
      exit_code=$exit_code_parallel
      echo "test fail in parallel on package (with zonal=${zonal}): " $package_name
    else
      echo "Passed test package in parallel (with zonal=${zonal}): " $package_name
    fi
  done
  return $exit_code
}

function print_test_logs() {
  readarray -t test_logs_array < "$TEST_LOGS_FILE"
  rm "$TEST_LOGS_FILE"
  for test_log_file in "${test_logs_array[@]}"
  do
    log_file=${test_log_file}
    if [ -f "$log_file" ]; then
      echo "=== Log for ${test_log_file} ==="
      cat "$log_file"
      echo "========================================="
    fi
  done
}

function run_e2e_tests_for_flat_bucket() {
  # Adding prefix `golang-grpc-test` to white list the bucket for grpc so that
  # we can run grpc related e2e tests.
  bucketPrefix="golang-grpc-test-gcsfuse-np-e2e-tests-"
  bucket_name_non_parallel=$(create_bucket $bucketPrefix)
  echo "Bucket name for non parallel tests: "$bucket_name_non_parallel
  echo ${bucket_name_non_parallel}>>"${bucketNamesFile}"

  bucketPrefix="golang-grpc-test-gcsfuse-p-e2e-tests-"
  bucket_name_parallel=$(create_bucket $bucketPrefix)
  echo "Bucket name for parallel tests: "$bucket_name_parallel
  echo ${bucket_name_parallel}>>"${bucketNamesFile}"

  echo "Running parallel tests..."
  run_parallel_tests TEST_DIR_PARALLEL $bucket_name_parallel &
  parallel_tests_pid=$!

  echo "Running non parallel tests ..."
  run_non_parallel_tests TEST_DIR_NON_PARALLEL $bucket_name_non_parallel &
  non_parallel_tests_pid=$!

  # Wait for all tests to complete.
  wait $parallel_tests_pid
  parallel_tests_exit_code=$?
  wait $non_parallel_tests_pid
  non_parallel_tests_exit_code=$?

  if [ $non_parallel_tests_exit_code != 0 ] || [ $parallel_tests_exit_code != 0 ];
  then
    return 1
  fi
  return 0
}

function run_e2e_tests_for_hns_bucket(){
   hns_bucket_name_parallel_group=$(create_hns_bucket)
   echo "Hns Bucket Created: "$hns_bucket_name_parallel_group
   echo ${hns_bucket_name_parallel_group}>>"${bucketNamesFile}"

   hns_bucket_name_non_parallel_group=$(create_hns_bucket)
   echo "Hns Bucket Created: "$hns_bucket_name_non_parallel_group
   echo ${hns_bucket_name_non_parallel_group}>>"${bucketNamesFile}"

   echo "Running tests for HNS bucket"
   run_parallel_tests TEST_DIR_PARALLEL "$hns_bucket_name_parallel_group" &
   parallel_tests_hns_group_pid=$!
   run_non_parallel_tests TEST_DIR_NON_PARALLEL "$hns_bucket_name_non_parallel_group" &
   non_parallel_tests_hns_group_pid=$!

   # Wait for all tests to complete.
   wait $parallel_tests_hns_group_pid
   parallel_tests_hns_group_exit_code=$?
   wait $non_parallel_tests_hns_group_pid
   non_parallel_tests_hns_group_exit_code=$?

   if [ $parallel_tests_hns_group_exit_code != 0 ] || [ $non_parallel_tests_hns_group_exit_code != 0 ];
   then
    return 1
   fi
   return 0
}

function run_e2e_tests_for_zonal_bucket(){
   zonal_bucket_name_parallel_group=$(create_zonal_bucket)
   echo "Zonal Bucket Created for parallel tests: "$zonal_bucket_name_parallel_group
   echo ${zonal_bucket_name_parallel_group}>>"${bucketNamesFile}"

   zonal_bucket_name_non_parallel_group=$(create_zonal_bucket)
   echo "Zonal Bucket Created for non-parallel tests: "$zonal_bucket_name_non_parallel_group
   echo ${zonal_bucket_name_non_parallel_group}>>"${bucketNamesFile}"

   echo "Running tests for ZONAL bucket"
   run_parallel_tests TEST_DIR_PARALLEL_FOR_ZB "$zonal_bucket_name_parallel_group" true &
   parallel_tests_zonal_group_pid=$!
   run_non_parallel_tests TEST_DIR_NON_PARALLEL_FOR_ZB "$zonal_bucket_name_non_parallel_group" true &
   non_parallel_tests_zonal_group_pid=$!

   # Wait for all tests to complete.
   wait $parallel_tests_zonal_group_pid
   parallel_tests_zonal_group_exit_code=$?
   wait $non_parallel_tests_zonal_group_pid
   non_parallel_tests_zonal_group_exit_code=$?

   if [ $parallel_tests_zonal_group_exit_code != 0 ] || [ $non_parallel_tests_zonal_group_exit_code != 0 ];
   then
    return 1
   fi
   return 0
}

function run_e2e_tests_for_tpc() {
  local bucket=$1
  if [ "$bucket" == "" ];
  then
    echo "Bucket name is required"
    return 1
  fi

  # Clean bucket before testing.
  gcloud --verbosity=error storage rm -r gs://"$bucket"/*

  # Run Operations e2e tests in TPC to validate all the functionality.
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/... --testOnTPCEndPoint=$RUN_TEST_ON_TPC_ENDPOINT $GO_TEST_SHORT_FLAG $PRESUBMIT_RUN_FLAG --zonal=false -p 1 --integrationTest -v --testbucket="$bucket" --testInstalledPackage=$RUN_E2E_TESTS_ON_PACKAGE $USE_PREBUILT_GCSFUSE_BINARY -timeout $INTEGRATION_TEST_TIMEOUT
  exit_code=$?

  set -e

  # Delete data after testing.
  gcloud --verbosity=error storage rm -r gs://"$bucket"/*

  if [ $exit_code != 0 ];
   then
     return 1
  fi
  return 0
}

function run_e2e_tests_for_emulator() {
  ./tools/integration_tests/emulator_tests/emulator_tests.sh $RUN_E2E_TESTS_ON_PACKAGE
}

function main(){
  # The name of a file containing the names of all the
  # buckets to be cleaned-up while exiting this program.
  bucketNamesFile=$(realpath ./bucketNames)"-"$(tr -dc 'a-z0-9' < /dev/urandom | head -c $RANDOM_STRING_LENGTH)
  # Delete all these buckets when the program exits.
  # Cleanup fuse build folder if created
  trap "cleanup_gcsfuse_once; delete_buckets_listed_in_file ${bucketNamesFile}" EXIT

  set -e

  upgrade_gcloud_version

  install_packages

  set +e

  # Decide whether to build GCSFuse based on RUN_E2E_TESTS_ON_PACKAGE
  if [ "$RUN_E2E_TESTS_ON_PACKAGE" != "true" ] && [ "$BUILD_BINARY_IN_SCRIPT" == "true" ]; then
    echo "RUN_E2E_TESTS_ON_PACKAGE is not 'true' (value: '${RUN_E2E_TESTS_ON_PACKAGE}') and BUILD_BINARY_IN_SCRIPT is 'true'. Building GCSFuse..."
    build_gcsfuse_once
    if [ $? -ne 0 ]; then
        echo "build_gcsfuse_once failed. Exiting."
        # The trap will handle cleanup
        exit 1
    fi

    USE_PREBUILT_GCSFUSE_BINARY="--gcsfuse_prebuilt_dir=${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}"
    echo "Script built GCSFuse at: ${BUILT_BY_SCRIPT_GCSFUSE_BUILD_DIR}"
  fi

  #run integration tests
  exit_code=0

  if ${RUN_TESTS_WITH_ZONAL_BUCKET}; then
    run_e2e_tests_for_zonal_bucket &
    e2e_tests_zonal_bucket_pid=$!
    wait $e2e_tests_zonal_bucket_pid
    e2e_tests_zonal_bucket_status=$?

    if [ $e2e_tests_zonal_bucket_status != 0 ]; then
      echo "The e2e tests for zonal bucket failed.."
      exit_code=1
    fi
  else
    # Run tpc test and exit in case RUN_TEST_ON_TPC_ENDPOINT is true.
    if [ "$RUN_TEST_ON_TPC_ENDPOINT" == true ]; then
         # Run tests for flat bucket
         run_e2e_tests_for_tpc gcsfuse-e2e-tests-tpc &
         e2e_tests_tpc_flat_bucket_pid=$!
         # Run tests for hns bucket
         run_e2e_tests_for_tpc gcsfuse-e2e-tests-tpc-hns &
         e2e_tests_tpc_hns_bucket_pid=$!

         wait $e2e_tests_tpc_flat_bucket_pid
         e2e_tests_tpc_flat_bucket_status=$?

         wait $e2e_tests_tpc_hns_bucket_pid
         e2e_tests_tpc_hns_bucket_status=$?

         if [ $e2e_tests_tpc_flat_bucket_status != 0 ];
         then
            echo "The e2e tests for flat bucket failed.."
            exit 1
         fi
         if [ $e2e_tests_tpc_hns_bucket_status != 0 ];
         then
             echo "The e2e tests for hns bucket failed.."
             exit 1
         fi
         # Exit to prevent the following code from executing for TPC.
         exit 0
    fi

    run_e2e_tests_for_hns_bucket &
    e2e_tests_hns_bucket_pid=$!

    run_e2e_tests_for_flat_bucket &
    e2e_tests_flat_bucket_pid=$!

    run_e2e_tests_for_emulator &
    e2e_tests_emulator_pid=$!

    wait $e2e_tests_emulator_pid
    e2e_tests_emulator_status=$?

    wait $e2e_tests_flat_bucket_pid
    e2e_tests_flat_bucket_status=$?

    wait $e2e_tests_hns_bucket_pid
    e2e_tests_hns_bucket_status=$?

    if [ $e2e_tests_flat_bucket_status != 0 ];
    then
      echo "The e2e tests for flat bucket failed.."
      exit_code=1
    fi

    if [ $e2e_tests_hns_bucket_status != 0 ];
    then
      echo "The e2e tests for hns bucket failed.."
      exit_code=1
    fi

    if [ $e2e_tests_emulator_status != 0 ];
    then
      echo "The e2e tests for emulator failed.."
      exit_code=1
    fi
  fi

  set -e

  print_test_logs

  exit $exit_code
}

#Main method to run script
main
