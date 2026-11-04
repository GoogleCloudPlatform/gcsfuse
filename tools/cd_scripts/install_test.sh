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

PYTHON_VERSION=3.11.9
INSTALL_PREFIX="$HOME/.local/python-$PYTHON_VERSION"

# Install common dependencies (Python 3.11, wget, tar)
# This is required for gcloud to function correctly (needs Python 3.10+) 
# and for the script to download/extract the SDK.
if command -v apt-get &> /dev/null; then
  sudo apt-get update -y > /dev/null
  sudo apt-get install -y \
      build-essential zlib1g-dev libncurses5-dev libgdbm-dev libnss3-dev \
      libssl-dev libreadline-dev libffi-dev curl libsqlite3-dev \
      libbz2-dev liblzma-dev tk-dev uuid-dev wget tar > /dev/null
elif command -v yum &> /dev/null; then
    # For RHEL-based systems, use 'yum' to install packages.
    # The "Development Tools" group is equivalent to 'build-essential' on Debian.
    # The '-devel' packages provide the necessary header files for compilation.
    sudo yum -y groupinstall "Development Tools" > /dev/null
    sudo yum -y install \
          zlib-devel ncurses-devel nss-devel openssl-devel \
          readline-devel libffi-devel curl sqlite-devel bzip2-devel \
          xz-devel tk-devel libuuid-devel wget > /dev/null
fi

# Download and build Python locally
cd /tmp
wget -q https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tgz
tar -xf Python-${PYTHON_VERSION}.tgz
cd Python-${PYTHON_VERSION}

echo "Configuring Python build for local install..."
./configure --enable-optimizations --prefix="$INSTALL_PREFIX" > /dev/null

echo "Building Python $PYTHON_VERSION..."
make -j"$(nproc)" > /dev/null

echo "Installing Python $PYTHON_VERSION locally at $INSTALL_PREFIX..."
make altinstall > /dev/null

echo "Python $PYTHON_VERSION installed at $INSTALL_PREFIX/bin/python3.11"
"$INSTALL_PREFIX/bin/python3.11" --version

# Ensure gcloud uses the newly installed Python 3.11
export CLOUDSDK_PYTHON="$INSTALL_PREFIX/bin/python3.11"

echo "Upgrade gcloud version"
gcloud version
wget -O gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz -q
sudo tar xzf gcloud.tar.gz && sudo cp -r google-cloud-sdk /usr/local && sudo rm -r google-cloud-sdk
sudo /usr/local/google-cloud-sdk/install.sh --quiet
export PATH=/usr/local/google-cloud-sdk/bin:$PATH
gcloud version && rm gcloud.tar.gz

# Helper function to fetch metadata value from GCE metadata server
function fetch_meta_data_value() {
  local metadata_key=$1
  curl -sfS -H "Metadata-Flavor: Google" "http://metadata.google.internal/computeMetadata/v1/instance/attributes/$metadata_key" 2>/dev/null || true
}

# Fetch configuration directly from instance metadata or environment
vm_instance_name=$(curl -sfS -H "Metadata-Flavor: Google" "http://metadata.google.internal/computeMetadata/v1/instance/name" 2>/dev/null || hostname)
to_release_version=$(fetch_meta_data_value "RELEASE_VERSION")
upload_bucket=$(fetch_meta_data_value "UPLOAD_BUCKET")
upload_bucket=${upload_bucket:-"gcsfuse-release-packages"}

# Backward compatibility fallback if RELEASE_VERSION is not present in metadata
if [[ -z "$to_release_version" ]]; then
  if gcloud storage cp "gs://${upload_bucket}/version-detail/details.txt" ./details.txt 2>/dev/null; then
    to_release_version=$(sed -n 1p details.txt | tr -d '\r\n')
  fi
fi

# Detect OS distribution via /etc/os-release
IS_DEBIAN_OR_UBUNTU=false
if [ -f /etc/os-release ]; then
  # shellcheck disable=SC1091
  . /etc/os-release
  distro_id="${ID:-}"
  distro_like="${ID_LIKE:-}"
  if [[ "$distro_id" =~ ^(ubuntu|debian)$ ]] || [[ "$distro_like" =~ (ubuntu|debian) ]]; then
    IS_DEBIAN_OR_UBUNTU=true
  fi
fi

touch ~/logs.txt

# Based on the OS type, run the installation commands
if [[ "$IS_DEBIAN_OR_UBUNTU" == true ]]; then
    if [[ "$to_release_version" == *"~beta"* ]]; then
         export GCSFUSE_REPO=gcsfuse-beta
    else
         export GCSFUSE_REPO="gcsfuse-$(lsb_release -c -s 2>/dev/null || echo 'bullseye')"
    fi

    # Don't use apt-key for Debian 11+ and Ubuntu 21+
    if { [[ $vm_instance_name == *"debian"* && !( "$vm_instance_name" < "release-test-debian-11") ]]; } || { [[ $vm_instance_name == *"ubuntu"* && !("$vm_instance_name" < "release-test-ubuntu-21") ]]; }; then
      echo "deb [signed-by=/usr/share/keyrings/cloud.google.asc] https://packages.cloud.google.com/apt $GCSFUSE_REPO main" | sudo tee /etc/apt/sources.list.d/gcsfuse.list
      curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo tee /usr/share/keyrings/cloud.google.asc >> ~/apt_key_logs.txt
    else
      echo "deb https://packages.cloud.google.com/apt $GCSFUSE_REPO main" | sudo tee /etc/apt/sources.list.d/gcsfuse.list
      curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add - >> ~/apt_key_logs.txt
    fi

    if grep -q -i warning ~/apt_key_logs.txt; then
      echo "Failure: Got warning while using apt-key" >> ~/logs.txt
    fi

    sudo apt-get update
    # Install to be released gcsfuse version (It can be a patch to older version so allow downgrades)
    sudo apt-get install -y --allow-downgrades gcsfuse="$to_release_version" >> ~/logs.txt
else
    # For RHEL and CentOS
    sudo yum install -y fuse
    if [[ "$to_release_version" == *"~beta"* ]]; then
        YUM_REPO_NAME=gcsfuse-el7-x86_64-beta
    else
        YUM_REPO_NAME=gcsfuse-el7-x86_64
    fi
    sudo tee /etc/yum.repos.d/gcsfuse.repo > /dev/null <<EOF
[gcsfuse]
name=gcsfuse (packages.cloud.google.com)
baseurl=https://packages.cloud.google.com/yum/repos/${YUM_REPO_NAME}
enabled=1
gpgcheck=1
repo_gpgcheck=0
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
      https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF
    # Attempt an install first, falling back to a standard downgrade if to-be-released is older version than already installed (patch releases).
    sudo yum install -y gcsfuse-"$to_release_version" || sudo yum downgrade -y gcsfuse-"$to_release_version" >> ~/logs.txt
fi

# Verify gcsfuse version (successful installation)
gcsfuse --version |& tee version.txt
installed_version=$(echo $(sed -n 1p version.txt) | cut -d' ' -f3)
if [[ "$installed_version" == "$to_release_version"* ]] || [[ "$to_release_version" == *"$installed_version"* ]]; then
    echo "GCSFuse to be released version installed correctly." &>> ~/logs.txt
else
    echo "Failure detected in to be released gcsfuse version installation: expected $to_release_version, got $installed_version" &>> ~/logs.txt
fi

# Uninstall gcsfuse and install old version.
if [[ "$IS_DEBIAN_OR_UBUNTU" == true ]]; then
  sudo apt-get remove -y gcsfuse |& tee -a ~/logs.txt
  sudo apt-get install -y gcsfuse=1.2.0 |& tee -a ~/logs.txt
else
  sudo yum -y remove gcsfuse |& tee -a ~/logs.txt
  sudo yum install -y gcsfuse-1.2.0 |& tee -a ~/logs.txt
fi

# Verify old version installation
gcsfuse --version |& tee version.txt
installed_version=$(echo $(sed -n 1p version.txt) | cut -d' ' -f3)
if [ "$installed_version" == "1.2.0" ]; then
  echo "GCSFuse old version (1.2.0) installed successfully" &>> ~/logs.txt
else
  echo "Failure detected in GCSFuse old version installation." &>> ~/logs.txt
fi

# Upgrade gcsfuse to latest version.
if [[ "$IS_DEBIAN_OR_UBUNTU" == true ]]; then
    sudo apt-get install --only-upgrade gcsfuse |& tee -a ~/logs.txt
else
    sudo yum -y upgrade gcsfuse |& tee -a ~/logs.txt
fi

# Verify that gcsfuse has been upgraded to the to_be_released version using version comparison.
gcsfuse --version |& tee version.txt
installed_version=$(echo $(sed -n 1p version.txt) | cut -d' ' -f3)
if [[ "$(printf '%s\n%s\n' "$to_release_version" "$installed_version" | sort -V | tail -n 1)" == "$installed_version" ]]; then
  echo "GCSFuse successfully upgraded to latest version: installed_version ($installed_version), to_release_version: ($to_release_version)" &>> ~/logs.txt
else
  echo "Failure detected in upgrading to latest gcsfuse version: installed_version ($installed_version), to_release_version: ($to_release_version)" &>> ~/logs.txt
fi

if grep -q Failure ~/logs.txt; then
    echo "Test failed" &>> ~/logs.txt
else
    touch success.txt
    gcloud storage cp success.txt "gs://${upload_bucket}/v${to_release_version}/installation-test/${vm_instance_name}/"
fi

gcloud storage cp ~/logs.txt "gs://${upload_bucket}/v${to_release_version}/installation-test/${vm_instance_name}/"
