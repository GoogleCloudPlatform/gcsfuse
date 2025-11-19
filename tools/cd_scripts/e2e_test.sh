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

# Conditionally install python3.11 and run gcloud installer with it for RHEL 8 and Rocky 8
INSTALL_COMMAND="sudo /usr/local/google-cloud-sdk/install.sh --quiet"
if [ -f /etc/os-release ]; then
    . /etc/os-release
    if [[ ($ID == "rhel" || $ID == "rocky") && $VERSION_ID == 8* ]]; then
        sudo yum install -y python311
        INSTALL_COMMAND="sudo env CLOUDSDK_PYTHON=/usr/bin/python3.11 /usr/local/google-cloud-sdk/install.sh --quiet"
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
wget -O go_tar.tar.gz https://go.dev/dl/go1.24.10.linux-${architecture}.tar.gz
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

# Create a temporary directory for artifacts
ARTIFACTS_DIR=$(mktemp -d)
export KOKORO_ARTIFACTS_DIR="$ARTIFACTS_DIR"
RUNTIME_STATS_FILE=$(mktemp)

echo "Artifacts directory: $ARTIFACTS_DIR"
echo "Runtime stats file: $RUNTIME_STATS_FILE"

IMPROVED_SCRIPT="./tools/integration_tests/improved_run_e2e_tests.sh"
COMMON_ARGS=(
  "--bucket-location" "$ZONE_NAME"
  "--skip-install"
  "--test-installed-package"
  "--runtime-stats-file" "$RUNTIME_STATS_FILE"
)

if [[ "$RUN_READ_CACHE_TESTS_ONLY" == "true" ]]; then
  echo "Running read cache tests only..."
  # Run Regional (HNS + Flat)
  $IMPROVED_SCRIPT "${COMMON_ARGS[@]}" --test-package read_cache --skip-emulator-tests
  
  # Run Zonal
  $IMPROVED_SCRIPT "${COMMON_ARGS[@]}" --zonal --test-package read_cache --skip-emulator-tests

elif [[ "$RUN_ON_ZB_ONLY" == "true" ]]; then
  echo "Running zonal tests only..."
  $IMPROVED_SCRIPT "${COMMON_ARGS[@]}" --zonal

else
  echo "Running all tests (Regional + Emulator)..."
  $IMPROVED_SCRIPT "${COMMON_ARGS[@]}"
fi

# Function to aggregate logs
aggregate_logs() {
  local bucket_type=$1
  local log_file=$2
  
  if [ -d "$ARTIFACTS_DIR/$bucket_type" ]; then
    find "$ARTIFACTS_DIR/$bucket_type" -name "sponge_log.log" | while read -r file; do
      echo "=== Log for $(basename $(dirname $file)) ===" >> "$log_file"
      cat "$file" >> "$log_file"
      echo "=========================================" >> "$log_file"
    done
  fi
}

declare -A failures
failures["flat"]=0
failures["hns"]=0
failures["zonal"]=0
failures["emulator"]=0

# Parse stats file
if [ -f "$RUNTIME_STATS_FILE" ]; then
  while read -r package bucket exit_code start end; do
    if [[ "$exit_code" != "0" ]]; then
      failures["$bucket"]=1
    fi
  done < "$RUNTIME_STATS_FILE"
fi

# Aggregate logs and upload
for bucket in flat hns zonal emulator; do
  target_log="$HOME/logs.txt"
  target_success="$HOME/success.txt"
  if [[ "$bucket" != "flat" ]]; then
    target_log="$HOME/logs-$bucket.txt"
    target_success="$HOME/success-$bucket.txt"
  fi
  
  aggregate_logs "$bucket" "$target_log"
  
  # Check if we have any stats for this bucket
  if grep -q " $bucket " "$RUNTIME_STATS_FILE"; then
    if [[ "${failures[$bucket]}" == "0" ]]; then
       touch "$target_success"
       gcloud storage cp "$target_success" gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
    else
       echo "Test failures detected in $bucket bucket." >> "$target_log"
    fi
    
    if [ -f "$target_log" ]; then
      gcloud storage cp "$target_log" gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
    fi
  fi
done

