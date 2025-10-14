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

# Install wget
if command -v apt-get &> /dev/null; then
    # For Debian/Ubuntu-based systems
    sudo apt-get update && sudo apt-get install -y wget
elif command -v yum &> /dev/null; then
    # For RHEL/CentOS-based systems
    sudo yum install -y wget
else
    exit 1
fi

# Upgrade gcloud
echo "Upgrade gcloud version"
wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
sudo tar xzf gcloud.tar.gz && sudo cp -r google-cloud-sdk /usr/local && sudo rm -r google-cloud-sdk

# Conditionally install python3.9 and run gcloud installer with it for RHEL 8 and Rocky 8
INSTALL_COMMAND="sudo /usr/local/google-cloud-sdk/install.sh --quiet"
if [ -f /etc/os-release ]; then
    . /etc/os-release
    if [[ ($ID == "rhel" || $ID == "rocky") && $VERSION_ID == 8* ]]; then
        sudo yum install -y python39
        INSTALL_COMMAND="sudo env CLOUDSDK_PYTHON=/usr/bin/python3.9 /usr/local/google-cloud-sdk/install.sh --quiet"
    fi
fi
$INSTALL_COMMAND 

export PATH=/usr/local/google-cloud-sdk/bin:$PATH
gcloud version && rm gcloud.tar.gz

# Extract the metadata parameters passed, for which we need the zone of the GCE VM
# on which the tests are supposed to run.
ZONE=$(curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/zone)
echo "Got ZONE=\"${ZONE}\" from metadata server."
# The format for the above extracted zone is projects/{project-id}/zones/{zone}, thus, from this
# need extracted zone name.
ZONE_NAME=$(basename "$ZONE")
# This parameter is passed as the GCE VM metadata at the time of creation.(Logic is handled in louhi stage script)
RUN_ON_ZB_ONLY=$(gcloud compute instances describe "$HOSTNAME" --zone="$ZONE_NAME" --format='get(metadata.run-on-zb-only)')
RUN_READ_CACHE_TESTS_ONLY=$(gcloud compute instances describe "$HOSTNAME" --zone="$ZONE_NAME" --format='get(metadata.run-read-cache-only)')
echo "RUN_ON_ZB_ONLY flag set to : \"${RUN_ON_ZB_ONLY}\""
echo "RUN_READ_CACHE_TESTS_ONLY flag set to : \"${RUN_READ_CACHE_TESTS_ONLY}\""

# Logging the tests being run on the active GCE VM
if [[ "$RUN_ON_ZB_ONLY" == "true" ]]; then
	echo "Running integration tests for Zonal bucket only..."
else
	echo "Running integration tests for non-zonal buckets only..."
fi

# Logging the tests being run on the active GCE VM
if [[ "$RUN_READ_CACHE_TESTS_ONLY" == "true" ]]; then
	echo "Running read cache test only..."
fi

#details.txt file contains the release version and commit hash of the current release.
gcloud storage cp gs://gcsfuse-release-packages/version-detail/details.txt .
# Writing VM instance name to details.txt (Format: release-test-<os-name>)
curl http://metadata.google.internal/computeMetadata/v1/instance/name -H "Metadata-Flavor: Google" >>details.txt

# Function to create the local user
create_user() {
  local USERNAME=$1
  local HOMEDIR=$2
  local DETAILS=$3
  if id "${USERNAME}" &>/dev/null; then
    echo "User ${USERNAME} already exists."
    return 0
  fi

  echo "Creating user ${USERNAME}..."
  if grep -qi -E 'ubuntu|debian' $DETAILS; then
    # For Ubuntu and Debian
    sudo adduser --disabled-password --home "${HOMEDIR}" --gecos "" "${USERNAME}"
  elif grep -qi -E 'rhel|centos|rocky' $DETAILS; then
    # For RHEL, CentOS, Rocky Linux
    sudo adduser --home-dir "${HOMEDIR}" "${USERNAME}" && sudo passwd -d "${USERNAME}"
  else
    echo "Unsupported OS type in details file." >&2
    return 1
  fi
  local exit_code=$?

  if [ ${exit_code} -eq 0 ]; then
    echo "User ${USERNAME} created successfully."
  else
    echo "Failed to create user ${USERNAME}." >&2
  fi
  return ${exit_code}
}

# Function to grant sudo access by creating a file in /etc/sudoers.d/
grant_sudo() {
  local USERNAME=$1
  local HOMEDIR=$2
  if ! id "${USERNAME}" &>/dev/null; then
    echo "User ${USERNAME} does not exist. Cannot grant sudo."
    return 1
  fi

  sudo mkdir -p /etc/sudoers.d/
  SUDOERS_FILE="/etc/sudoers.d/${USERNAME}"

  if sudo test -f "${SUDOERS_FILE}"; then
    echo "Sudoers file ${SUDOERS_FILE} already exists."
  else
    echo "Granting ${USERNAME} NOPASSWD sudo access..."
    # Create the sudoers file with the correct content
    if ! echo "${USERNAME} ALL=(ALL:ALL) NOPASSWD:ALL" | sudo tee "${SUDOERS_FILE}" > /dev/null; then
      echo "Failed to create sudoers file." >&2
      return 1
    fi

    # Set the correct permissions on the sudoers file
    if ! sudo chmod 440 "${SUDOERS_FILE}"; then
      echo "Failed to set permissions on sudoers file." >&2
      # Attempt to clean up the partially created file
      sudo rm -f "${SUDOERS_FILE}"
      return 1
    fi
    echo "Sudo access granted to ${USERNAME} via ${SUDOERS_FILE}."
  fi
  return 0
}
################################################################################
# Main script execution flow starts here.
# The script will first attempt to create the user specified by $USERNAME.
# If the user creation is successful, it will then proceed to grant sudo
# privileges to the newly created user.
################################################################################
USERNAME=starterscriptuser
HOMEDIR="/home/${USERNAME}"
DETAILS_FILE=$(pwd)/details.txt

create_user $USERNAME $HOMEDIR $DETAILS_FILE
grant_sudo  $USERNAME $HOMEDIR


# Run the following as starterscriptuser
sudo -u starterscriptuser bash -c '
# Exit immediately if a command exits with a non-zero status.
set -e
# Print commands and their arguments as they are executed.
set -x

# Since we are now operating as the starterscriptuser, we need to set the environment variable for this user again.
export PATH=/usr/local/google-cloud-sdk/bin:$PATH

# Export the RUN_ON_ZB_ONLY variable so that it is available in the environment of the 'starterscriptuser' user.
# Since we are running the subsequent script as 'starterscriptuser' using sudo, the environment of 'starterscriptuser'
# would not automatically have access to the environment variables set by the original user (i.e. $RUN_ON_ZB_ONLY).
# By exporting this variable, we ensure that the value of RUN_ON_ZB_ONLY is passed into the 'starterscriptuser' script
# and can be used for conditional logic or decisions within that script.
export RUN_ON_ZB_ONLY='$RUN_ON_ZB_ONLY'
export RUN_READ_CACHE_TESTS_ONLY='$RUN_READ_CACHE_TESTS_ONLY'

#Copy details.txt to starterscriptuser home directory and create logs.txt
cd ~/
cp /details.txt .
touch logs.txt
touch logs-hns.txt
touch logs-zonal.txt
LOG_FILE='~/logs.txt'

if [[ "$RUN_ON_ZB_ONLY" == "true" ]]; then
  LOG_FILE='~/logs-zonal.txt'
fi

echo "User: $USER" &>> ${LOG_FILE}
echo "Current Working Directory: $(pwd)"  &>> ${LOG_FILE}


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
    gcloud storage cp gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/gcsfuse_$(sed -n 1p details.txt)_${architecture}.deb .
    sudo dpkg -i gcsfuse_$(sed -n 1p details.txt)_${architecture}.deb |& tee -a ${LOG_FILE}

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
    uname=$(uname -m)

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
    gcloud storage cp gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/gcsfuse-$(sed -n 1p details.txt)-1.${uname}.rpm .
    sudo yum -y localinstall gcsfuse-$(sed -n 1p details.txt)-1.${uname}.rpm

    #install wget
    sudo yum -y install wget

    #install git
    sudo yum -y install git

    #install Development tools
    sudo yum -y install gcc gcc-c++ make
fi

# install go
wget -O go_tar.tar.gz https://go.dev/dl/go1.24.5.linux-${architecture}.tar.gz
sudo tar -C /usr/local -xzf go_tar.tar.gz
export PATH=${PATH}:/usr/local/go/bin
#Write gcsfuse and go version to log file
gcsfuse --version |& tee -a ${LOG_FILE}
go version |& tee -a ${LOG_FILE}

# Clone and checkout gcsfuse repo
export PATH=${PATH}:/usr/local/go/bin
git clone https://github.com/googlecloudplatform/gcsfuse |& tee -a ${LOG_FILE}
cd gcsfuse
bash ./perfmetrics/scripts/install_latest_gcloud.sh

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

git checkout $(sed -n 2p ~/details.txt) |& tee -a ${LOG_FILE}

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
  "kernel_list_cache"
  "concurrent_operations"
  "mount_timeout"
  "stale_handle"
  "negative_stat_cache"
  "streaming_writes"
  "release_version"
  "readdirplus"
  "dentry_cache"
  "buffered_read"
  # Disabled because of b/451462914.
  #"requester_pays_bucket"
  "flag_optimizations"
)

# These tests never become parallel as they are changing bucket permissions.
TEST_DIR_NON_PARALLEL=(
  "readonly"
  "managed_folders"
  "readonly_creds"
  "list_large_dir"
)

# For Zonal buckets : Test directory arrays
TEST_DIR_PARALLEL_ZONAL=(
  buffered_read
  concurrent_operations
  dentry_cache
  explicit_dir
  flag_optimizations
  gzip
  implicit_dir
  interrupt
  kernel_list_cache
  local_file
  log_rotation
  monitoring
  mount_timeout
  mounting
  negative_stat_cache
  operations
  rapid_appends
  read_large_files
  readdirplus
  release_version
  rename_dir_limit
  stale_handle
  streaming_writes
  unfinalized_object
  write_large_files
)

# For Zonal Buckets :  These tests never become parallel as they are changing bucket permissions.
TEST_DIR_NON_PARALLEL_ZONAL=(
  "managed_folders"
  "readonly"
  "readonly_creds"
  "list_large_dir"
)

# Create a temporary file to store the log file name.
TEST_LOGS_FILE=$(mktemp)

INTEGRATION_TEST_TIMEOUT=240m

# This method runs test packages in sequence.Necessary when the tests involves
# permissions modification etc.
# Arguments:
#   $1: BUCKET_NAME (The name of the GCS bucket to use for tests.)
#   $2: IS_ZONAL_BUCKET_FLAG (Boolean flag: 'true' if the bucket is zonal, 'false' otherwise.)
#   $3: NAME_OF_TEST_DIR_ARRAY (The shell variable name of the array containing test directory.)
function run_non_parallel_tests() {
  if [ "$#" -ne 3 ]; then
    echo "Incorrect number of arguments passed, Expecting <BUCKET_NAME>
    <IS_ZONAL_BUCKET> <TEST_DIR_ARRAY>"
    exit 1
  fi
  local exit_code=0 # Initialize to 0 for success
  local BUCKET_NAME=$1
  local zonal=$2
  if [[ -z $3 ]]; then
    return 1 # The name of the test array cannot be empty.
  fi
  local -n test_array=$3 # Create a nameref to this array.

  for test_dir_np in "${test_array[@]}"
  do
    test_path_non_parallel="./tools/integration_tests/$test_dir_np"
    # To make it clear whether tests are running on a flat or HNS or zonal bucket, We kept the log file naming
    # convention to include the bucket name as a suffix (e.g., package_name_bucket_name).
    local log_file="/tmp/${test_dir_np}_${BUCKET_NAME}.log"
    echo "$log_file" >> "$TEST_LOGS_FILE" # Use double quotes for log_file
    GODEBUG=asyncpreemptoff=1 go test "$test_path_non_parallel" -p 1 --zonal="${zonal}" --integrationTest -v --testbucket="$BUCKET_NAME" --testInstalledPackage=true -timeout "$INTEGRATION_TEST_TIMEOUT" > "$log_file" 2>&1
    exit_code_non_parallel=$?
    if [ $exit_code_non_parallel -ne 0 ]; then
      exit_code=$exit_code_non_parallel
    fi
  done
  return $exit_code
}

#This method runs test packages in parallel.
# Arguments:
#   $1: BUCKET_NAME (The name of the GCS bucket to use for tests.)
#   $2: IS_ZONAL_BUCKET_FLAG (Boolean flag: 'true' if the bucket is zonal, 'false' otherwise.)
#   $3: NAME_OF_TEST_DIR_ARRAY (The shell variable name of the array containing test directory.)
function run_parallel_tests() {
  if [ "$#" -ne 3 ]; then
    echo "Incorrect number of arguments passed, Expecting <BUCKET_NAME>
    <IS_ZONAL_BUCKET> <TEST_DIR_ARRAY>"
    exit 1
  fi
  local exit_code=0
  local BUCKET_NAME=$1
  local zonal=$2
  if [[ -z $3 ]]; then
    return 1 # The name of the test array cannot be empty.
  fi
  local -n test_array=$3 # Create a nameref to this array.
  local pids=()

  for test_dir_p in "${test_array[@]}"
  do
    test_path_parallel="./tools/integration_tests/$test_dir_p"
    # To make it clear whether tests are running on a flat or HNS or zonal bucket, We kept the log file naming
    # convention to include the bucket name as a suffix (e.g., package_name_bucket_name).
    local log_file="/tmp/${test_dir_p}_${BUCKET_NAME}.log"
    echo "$log_file" >> "$TEST_LOGS_FILE"
    GODEBUG=asyncpreemptoff=1 go test "$test_path_parallel" -p 1 --zonal="${zonal}" --integrationTest -v --testbucket="$BUCKET_NAME" --testInstalledPackage=true -timeout "$INTEGRATION_TEST_TIMEOUT" > "$log_file" 2>&1 &
    pid=$!
    pids+=("$pid")
  done
  for pid in "${pids[@]}"; do
    wait "$pid"
    exit_code_parallel=$?
    if [ $exit_code_parallel -ne 0 ]; then
      exit_code=$exit_code_parallel
    fi
  done
  return $exit_code
}

#Common method to invoke e2e tests on different types of buckets: flat, HNS or Zonal
# Arguments:
#   $1: BUCKET-TYPE (flat/hns/zonal)
#   $2: TEST_DIR_PARALLEL (list of test packages that can be run in parallel)
#   $3: TEST_DIR_NON_PARALLEL (list of test packages that should be run in sequence)
#   $4: IS_ZONAL_BUCKET_FLAG (Boolean flag: 'true' if the bucket is zonal, 'false' otherwise.)
function run_e2e_tests() {
  if [ "$#" -ne 4 ]; then
    echo "Incorrect number of arguments passed, Expecting <TESTCASE>
    <NAME_OF_PARALLEL_TEST_DIR_ARRAY> <NAME_OF_PARALLEL_TEST_DIR_ARRAY>
    <IS_ZONAL_BUCKET_FLAG>"
    exit 1
  fi
  local testcase=$1
  local -n test_dir_parallel=$2
  local -n test_dir_non_parallel=$3
  local is_zonal=$4
  local overall_exit_code=0

  prefix=$(sed -n 3p ~/details.txt)
  if [[ "$testcase" != "flat" ]]; then
    prefix=$(sed -n 3p ~/details.txt)-$testcase
  fi

  local bkt_non_parallel=$prefix
  echo "Bucket name to run non-parallel tests sequentially: $bkt_non_parallel"

  local bkt_parallel=$prefix-parallel
  echo "Bucket name to run parallel tests: $bkt_parallel"

  echo "Running parallel tests..."
  run_parallel_tests  "$bkt_parallel" "$is_zonal" test_dir_parallel & # Pass the name of the array
  parallel_tests_pid=$!

  echo "Running non parallel tests ..."
  run_non_parallel_tests  "$bkt_non_parallel" "$is_zonal" test_dir_non_parallel & # Pass the name of the array
  non_parallel_tests_pid=$!

  wait "$parallel_tests_pid"
  local parallel_tests_exit_code=$?
  wait "$non_parallel_tests_pid"
  local non_parallel_tests_exit_code=$?

  if [ "$non_parallel_tests_exit_code" -ne 0 ]; then
    overall_exit_code=$non_parallel_tests_exit_code
  fi

  if [ "$parallel_tests_exit_code" -ne 0 ]; then
    overall_exit_code=$parallel_tests_exit_code
  fi
  return $overall_exit_code
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
      elif [[ "$test_log_file" == *"zonal"* ]]; then
        output_file="$HOME/logs-zonal.txt"
      else
        output_file="$HOME/logs.txt"
      fi

      echo "=== Log for ${test_log_file} ===" >> "$output_file"
      cat "$log_file" >> "$output_file"
      echo "=========================================" >> "$output_file"
    fi
  done
}

# Function to log test results and upload them to GCS based on exit status.
# Arguments: $1 = name of the associative array containing testcase exit statuses.
function log_based_on_exit_status() {
  if [ "$#" -ne 1 ]; then
    echo "Incorrect number of arguments passed, Expecting <EXIT_STATUS_ARRAY_NAME>"
    exit 1
  fi
  gather_test_logs
  local -n exit_status_array=$1

  for testcase in "${!exit_status_array[@]}"
    do
        local logfile=""
        local successfile=""
        if [[ "$testcase" == "flat" ]]; then
          logfile="$HOME/logs.txt"
          successfile="$HOME/success.txt"
        else
          logfile="$HOME/logs-$testcase.txt"
          successfile="$HOME/success-$testcase.txt"
        fi
        if [ "${exit_status_array["$testcase"]}" != 0 ];
        then
            echo "Test failures detected in $testcase bucket." &>> $logfile
        else
            touch $successfile
            gcloud storage cp $successfile gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
        fi
    gcloud storage cp $logfile gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
    done

}

# Function to run emulator-based E2E tests and log results.
function run_e2e_tests_for_emulator_and_log() {
  ./tools/integration_tests/emulator_tests/emulator_tests.sh true > ~/logs-emulator.txt
  emulator_test_status=$?
  if [ $e2e_tests_emulator_status != 0 ];
    then
        echo "Test failures detected in emulator based tests." &>> ~/logs-emulator.txt
    else
        touch success-emulator.txt
        gcloud storage cp success-emulator.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
    fi
    gcloud storage cp ~/logs-emulator.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
}

# Declare an associative array to store the exit status of different test runs.
declare -A exit_status
if [[ "$RUN_READ_CACHE_TESTS_ONLY" == "true" ]]; then
    read_cache_test_dir_parallel=() # Empty for read cache tests only
    read_cache_test_dir_non_parallel=("read_cache")

    # Run E2E tests for flat, HNS, and zonal buckets with only read cache tests.
    # Running sequentially due to known limitations of simultaneous execution.
    run_e2e_tests "flat" read_cache_test_dir_parallel read_cache_test_dir_non_parallel false
    exit_status["flat"]=$?

    run_e2e_tests "hns" read_cache_test_dir_parallel read_cache_test_dir_non_parallel false
    exit_status["hns"]=$?

    run_e2e_tests "zonal" read_cache_test_dir_parallel read_cache_test_dir_non_parallel true
    exit_status["zonal"]=$?
else
    # If not running *only* read cache tests, proceed with full test suites.
    if [[ "$RUN_ON_ZB_ONLY" == "true" ]]; then
        # If only zonal bucket tests are to be run.
        run_e2e_tests "zonal" TEST_DIR_PARALLEL_ZONAL TEST_DIR_NON_PARALLEL_ZONAL true
        exit_status["zonal"]=$?
    else
        # Run flat and HNS tests concurrently in the background.
        run_e2e_tests "flat" TEST_DIR_PARALLEL TEST_DIR_NON_PARALLEL false &
        flat_test_pid=$!

        run_e2e_tests "hns" TEST_DIR_PARALLEL TEST_DIR_NON_PARALLEL false &
        hns_test_pid=$!

        # Wait for PIDs and populate exit_status associative array
        wait $flat_test_pid
        exit_status["flat"]=$?

        wait $hns_test_pid
        exit_status["hns"]=$?

        # Run emulator tests and log their results.
        run_e2e_tests_for_emulator_and_log
    fi

fi
#Log results based on the collected exit statuses.
log_based_on_exit_status exit_status

'
