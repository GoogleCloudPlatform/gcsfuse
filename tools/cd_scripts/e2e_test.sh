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
gcloud version
wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
sudo tar xzf gcloud.tar.gz && sudo cp -r google-cloud-sdk /usr/local && sudo rm -r google-cloud-sdk
sudo /usr/local/google-cloud-sdk/install.sh
export PATH=/usr/local/google-cloud-sdk/bin:$PATH
gcloud version && rm gcloud.tar.gz

#details.txt file contains the release version and commit hash of the current release.
gcloud storage cp gs://gcsfuse-release-packages/version-detail/details.txt .
# Writing VM instance name to details.txt (Format: release-test-<os-name>)
curl http://metadata.google.internal/computeMetadata/v1/instance/name -H "Metadata-Flavor: Google" >> details.txt

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

#Copy details.txt to starterscriptuser home directory and create logs.txt
cd ~/
cp /details.txt .
touch logs.txt

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
    gcloud storage cp gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/gcsfuse_$(sed -n 1p details.txt)_${architecture}.deb .
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
wget -O go_tar.tar.gz https://go.dev/dl/go1.24.11.linux-${architecture}.tar.gz
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
# TODO: Running both tests in parallel leads to more test failures. Since Louhi runs on a smaller machine, this does not significantly reduce execution time.
# We can include this task as part of the parallel e2e test project in the Louhi pipeline.

# Run tests on HNS bucket
GODEBUG=asyncpreemptoff=1 CGO_ENABLED=0 go test ./tools/integration_tests/... -p 1 -short --integrationTest -v --testbucket=$(sed -n 3p ~/details.txt)-hns --timeout=60m &>> ~/logs-hns.txt
if [ $? -ne 0 ];
then
    echo "Test failures detected" &>> ~/logs-hns.txt
else
    touch success-hns.txt
    gsutil cp success-hns.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
fi
gsutil cp ~/logs-hns.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/

# Run tests on FLAT bucket
GODEBUG=asyncpreemptoff=1 CGO_ENABLED=0 go test ./tools/integration_tests/... -p 1 -short --integrationTest -v --testbucket=$(sed -n 3p ~/details.txt) --timeout=60m &>> ~/logs.txt
if [ $? -ne 0 ];
then
    echo "Test failures detected" &>> ~/logs.txt
else
    touch success.txt
    gsutil cp success.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
fi
gsutil cp ~/logs.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
'
