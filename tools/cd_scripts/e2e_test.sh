#! /bin/bash
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

# Print commands and their arguments as they are executed.
set -x
# Exit immediately if a command exits with a non-zero status.
set -e

#details.txt file contains the release version and commit hash of the current release.
gsutil cp  gs://gcsfuse-release-packages/version-detail/details.txt .
# Writing VM instance name to details.txt (Format: release-test-<os-name>)
curl http://metadata.google.internal/computeMetadata/v1/instance/name -H "Metadata-Flavor: Google" >> details.txt

# Based on the os type(from vm instance name) in detail.txt, run the following commands to add starterscriptuser
if grep -q ubuntu details.txt || grep -q debian details.txt;
then
#  For ubuntu and debian os
    sudo adduser --ingroup google-sudoers --disabled-password --home=/home/starterscriptuser --gecos "" starterscriptuser
else
#  For rhel and centos
    sudo adduser -g google-sudoers --home-dir=/home/starterscriptuser starterscriptuser
fi

# Run the following as starterscriptuser
sudo -u starterscriptuser bash -c '
# Exit immediately if a command exits with a non-zero status.
set -e
# Print commands and their arguments as they are executed.
set -x

#Copy details.txt to starterscriptuser home directory and create logs.txt
cd ~/
cp /details.txt .
touch logs.txt
touch logs-hns.txt

echo User: $USER &>> ~/logs.txt
echo Current Working Directory: $(pwd)  &>> ~/logs.txt

# Based on the os type in detail.txt, run the following commands for setup

if grep -q ubuntu details.txt || grep -q debian details.txt;
then
#  For Debian and Ubuntu os
    # architecture can be amd64 or arm64
    architecture=$(dpkg --print-architecture)

    sudo apt update

    #Install fuse
    sudo apt install -y fuse

    # download and install gcsfuse deb package
    gsutil cp gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/gcsfuse_$(sed -n 1p details.txt)_${architecture}.deb .
    sudo dpkg -i gcsfuse_$(sed -n 1p details.txt)_${architecture}.deb |& tee -a ~/logs.txt

    # install wget
    sudo apt install -y wget

    #install git
    sudo apt install -y git

   # install python3-setuptools tools.
   sudo apt-get install -y gcc python3-dev python3-setuptools
   # Downloading composite object requires integrity checking with CRC32c in gsutil.
   # it requires to install crcmod.
   sudo apt install -y python3-crcmod

    #install build-essentials
    sudo apt install -y build-essential
else
#  For rhel and centos
    # uname can be aarch or x86_64
    uname=$(uname -i)

    if [[ $uname == "x86_64" ]]; then
      architecture="amd64"
    elif [[ $uname == "aarch64" ]]; then
      architecture="arm64"
    fi

    sudo yum makecache
    sudo yum -y update

    #Install fuse
    sudo yum -y install fuse

    #download and install gcsfuse rpm package
    gsutil cp gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/gcsfuse-$(sed -n 1p details.txt)-1.${uname}.rpm .
    sudo yum -y localinstall gcsfuse-$(sed -n 1p details.txt)-1.${uname}.rpm

    #install wget
    sudo yum -y install wget

    #install git
    sudo yum -y install git

    #install Development tools
    sudo yum -y install gcc gcc-c++ make
fi

# install go
wget -O go_tar.tar.gz https://go.dev/dl/go1.24.9.linux-${architecture}.tar.gz
sudo tar -C /usr/local -xzf go_tar.tar.gz
export PATH=${PATH}:/usr/local/go/bin
#Write gcsfuse and go version to log file
gcsfuse --version |& tee -a ~/logs.txt
go version |& tee -a ~/logs.txt

# Clone and checkout gcsfuse repo
export PATH=${PATH}:/usr/local/go/bin
git clone https://github.com/googlecloudplatform/gcsfuse |& tee -a ~/logs.txt
cd gcsfuse

# Installation of crcmod is working through pip only on rhel and centos.
# For debian and ubuntu, we are installing through sudo apt.
if grep -q rhel details.txt || grep -q centos details.txt;
then
    # install python3-setuptools tools and python3-pip
    sudo yum -y install gcc python3-devel python3-setuptools redhat-rpm-config
    sudo yum -y install python3-pip
    # Downloading composite object requires integrity checking with CRC32c in gsutil.
    # it requires to install crcmod.
    pip3 install --require-hashes -r tools/cd_scripts/requirements.txt --user
fi

git checkout $(sed -n 2p ~/details.txt) |& tee -a ~/logs.txt

#run tests with testbucket flag
set +e
# Test directory arrays
TEST_DIR_PARALLEL=(
  "monitoring"
  "local_file"
  "log_rotation"
  "mounting"
  "gzip"
  "write_large_files"
  "rename_dir_limit"
  "read_large_files"
  "explicit_dir"
  "implicit_dir"
  "interrupt"
  "operations"
  "log_content"
  "kernel_list_cache"
  "concurrent_operations"
  "mount_timeout"
  "stale_handle"
  "stale_handle_streaming_writes"
  "negative_stat_cache"
  "streaming_writes"
)

# These tests never become parallel as they are changing bucket permissions.
TEST_DIR_NON_PARALLEL=(
  "readonly"
  "managed_folders"
  "readonly_creds"
  "list_large_dir"
  "read_cache"
)

# Create a temporary file to store the log file name.
TEST_LOGS_FILE=$(mktemp)

INTEGRATION_TEST_TIMEOUT=240m

function run_non_parallel_tests() {
  local exit_code=0
  local -n test_array=$1
  local BUCKET_NAME=$2
  for test_dir_np in "${test_array[@]}"
  do
    test_path_non_parallel="./tools/integration_tests/$test_dir_np"
    # To make it clear whether tests are running on a flat or HNS bucket, We kept the log file naming
    # convention to include the bucket name as a suffix (e.g., package_name_bucket_name).
    local log_file="/tmp/${test_dir_np}_${BUCKET_NAME}.log"
    echo $log_file >> $TEST_LOGS_FILE
    # Executing integration tests
    GODEBUG=asyncpreemptoff=1 go test $test_path_non_parallel -p 1 --integrationTest -v --testbucket=$BUCKET_NAME --testInstalledPackage=true -timeout $INTEGRATION_TEST_TIMEOUT > "$log_file" 2>&1
    exit_code_non_parallel=$?
    if [ $exit_code_non_parallel != 0 ]; then
      exit_code=$exit_code_non_parallel
    fi
  done
  return $exit_code
}

function run_parallel_tests() {
  local exit_code=0
  local -n test_array=$1
  local BUCKET_NAME=$2
  local pids=()
  local benchmark_flags=""

  for test_dir_p in "${test_array[@]}"
  do
    # Unlike regular tests,benchmark tests are not executed by default when using go test .
    # The -bench flag yells go test to run the benchmark tests and report their results by
    # enabling the benchmarking framework.
    if [ $test_dir_p == "benchmarking" ]; then
        benchmark_flags="-bench=."
    fi
    test_path_parallel="./tools/integration_tests/$test_dir_p"
    # To make it clear whether tests are running on a flat or HNS bucket, We kept the log file naming
    # convention to include the bucket name as a suffix (e.g., package_name_bucket_name).
    local log_file="/tmp/${test_dir_p}_${BUCKET_NAME}.log"
    echo $log_file >> $TEST_LOGS_FILE
    # Executing integration tests
    GODEBUG=asyncpreemptoff=1 go test $test_path_parallel $benchmark_flags -p 1 --integrationTest -v --testbucket=$BUCKET_NAME --testInstalledPackage=true -timeout $INTEGRATION_TEST_TIMEOUT > "$log_file" 2>&1 &
    pid=$!  # Store the PID of the background process
    pids+=("$pid")  # Optionally add the PID to an array for later
  done
  # Wait for processes and collect exit codes
  for pid in "${pids[@]}"; do
    wait $pid
    exit_code_parallel=$?
    if [ $exit_code_parallel != 0 ]; then
      exit_code=$exit_code_parallel
    fi
  done
  return $exit_code
}

function run_e2e_tests_for_flat_bucket() {
  flat_bucket_name_non_parallel=$(sed -n 3p ~/details.txt)
  echo "Flat Bucket name to run tests sequentially: "$flat_bucket_name_non_parallel

  flat_bucket_name_parallel=$(sed -n 3p ~/details.txt)-parallel
  echo "Flat Bucket name to run tests parallelly: "$flat_bucket_name_parallel

  echo "Running parallel tests..."
  run_parallel_tests TEST_DIR_PARALLEL "$flat_bucket_name_parallel" &
  parallel_tests_pid=$!

 echo "Running non parallel tests ..."
 run_non_parallel_tests TEST_DIR_NON_PARALLEL "$flat_bucket_name_non_parallel" &
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
  hns_bucket_name_non_parallel=$(sed -n 3p ~/details.txt)-hns
  echo "HNS Bucket name to run tests sequentially: "$hns_bucket_name_non_parallel

  hns_bucket_name_parallel=$(sed -n 3p ~/details.txt)-hns-parallel
  echo "HNS Bucket name to run tests parallelly: "$hns_bucket_name_parallel

   echo "Running tests for HNS bucket"
   run_parallel_tests TEST_DIR_PARALLEL "$hns_bucket_name_parallel" &
   parallel_tests_hns_group_pid=$!
   run_non_parallel_tests TEST_DIR_NON_PARALLEL "$hns_bucket_name_non_parallel" &
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

function run_e2e_tests_for_emulator() {
  ./tools/integration_tests/emulator_tests/emulator_tests.sh true > ~/logs-emulator.txt
}

function gather_test_logs() {
  readarray -t test_logs_array < "$TEST_LOGS_FILE"
  rm "$TEST_LOGS_FILE"
  for test_log_file in "${test_logs_array[@]}"
  do
    log_file=${test_log_file}
    if [ -f "$log_file" ]; then
      if [[ "$test_log_file" == *"hns"* ]]; then
        output_file="$HOME/logs-hns.txt"
      else
        output_file="$HOME/logs.txt"
      fi

      echo "=== Log for ${test_log_file} ===" >> "$output_file"
      cat "$log_file" >> "$output_file"
      echo "=========================================" >> "$output_file"
    fi
  done
}

echo "Running integration tests for HNS bucket..."
run_e2e_tests_for_hns_bucket &
e2e_tests_hns_bucket_pid=$!

echo "Running integration tests for FLAT bucket..."
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

gather_test_logs

if [ $e2e_tests_flat_bucket_status != 0 ]
then
    echo "Test failures detected in FLAT bucket." &>> ~/logs.txt
else
    touch success.txt
    gsutil cp success.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
fi
gsutil cp ~/logs.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/

if [ $e2e_tests_hns_bucket_status != 0 ];
then
    echo "Test failures detected in HNS bucket." &>> ~/logs-hns.txt
else
    touch success-hns.txt
    gsutil cp success-hns.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
fi
gsutil cp ~/logs-hns.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/

if [ $e2e_tests_emulator_status != 0 ];
then
    echo "Test failures detected in emulator based tests." &>> ~/logs-emulator.txt
else
    touch success-emulator.txt
    gsutil cp success-emulator.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
fi

gsutil cp ~/logs-emulator.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
'
