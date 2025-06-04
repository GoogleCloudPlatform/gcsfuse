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

#details.txt file contains the release version and commit hash of the current release.
gsutil cp  gs://gcsfuse-release-packages/version-detail/details.txt .
# Writing VM instance name to details.txt (Format: release-test-<os-name>)
vm_instance_name=$(curl http://metadata.google.internal/computeMetadata/v1/instance/name -H "Metadata-Flavor: Google")
# first line of details.txt contains the release version in the format MAJOR.MINOR.PATCH
to_release_version=$(sed '1q' details.txt | tr -d '\n')
echo $vm_instance_name >> details.txt
touch ~/logs.txt

# Based on the os type(from vm instance name) in detail.txt, run the following
# commands to install gcsfuse.
if grep -q ubuntu details.txt || grep -q debian details.txt;
then
    if grep -q "~beta" details.txt;
    then
         export GCSFUSE_REPO=gcsfuse-beta
    else
         export GCSFUSE_REPO=gcsfuse-`lsb_release -c -s`
    fi
    #  For ubuntu and debian os

    # Don't use apt-key for Debian 11+ and Ubuntu 21+
    if { [[ $vm_instance_name == *"debian"*  &&  !( "$vm_instance_name" < "release-test-debian-11") ]]; } || { [[ $vm_instance_name == *"ubuntu"*  && !("$vm_instance_name" < "release-test-ubuntu-21") ]]; }
    then
      echo "deb [signed-by=/usr/share/keyrings/cloud.google.asc] https://packages.cloud.google.com/apt $GCSFUSE_REPO main" | sudo tee /etc/apt/sources.list.d/gcsfuse.list
      curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo tee /usr/share/keyrings/cloud.google.asc >> ~/apt_key_logs.txt
    else
      echo "deb https://packages.cloud.google.com/apt $GCSFUSE_REPO main" | sudo tee /etc/apt/sources.list.d/gcsfuse.list
      curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add - >> ~/apt_key_logs.txt
    fi

    if grep -q -i warning ~/apt_key_logs.txt;
    then
      echo "Failure: Got warning while using apt-key" >> ~/logs.txt
    fi

    sudo apt-get update
    # Install to be released gcsfuse version
    sudo apt-get install -y gcsfuse="$to_release_version" >> ~/logs.txt
else
#  For rhel and centos
    sudo yum install fuse
    if grep -q "~beta" details.txt;
    then
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
sudo yum install -y gcsfuse-"$to_release_version" >> ~/logs.txt
fi

# Verify gcsfuse version (successful installation)
gcsfuse --version |& tee version.txt
installed_version=$(echo $(sed -n 1p version.txt) | cut -d' ' -f3)
if grep -q $installed_version details.txt; then
    echo "GCSFuse to be released version installed correctly." &>> ~/logs.txt
else
    echo "Failure detected in to be released gcsfuse version installation." &>> ~/logs.txt
fi


# Uninstall gcsfuse and install old version.
if grep -q ubuntu details.txt || grep -q debian details.txt;
then
  sudo apt-get remove -y gcsfuse |& tee -a ~/logs.txt
  sudo apt-get install -y gcsfuse=1.2.0 |& tee -a ~/logs.txt
else
  sudo yum -y remove gcsfuse |& tee -a ~/logs.txt
  sudo yum install -y gcsfuse-1.2.0 |& tee -a ~/logs.txt
fi

# verify old version installation
gcsfuse --version |& tee version.txt
installed_version=$(echo $(sed -n 1p version.txt) | cut -d' ' -f3)
if [ $installed_version == "1.2.0" ]; then
  echo "GCSFuse old version (1.2.0) installed successfully" &>> ~/logs.txt
else
  echo "Failure detected in GCSFuse old version installation." &>> ~/logs.txt
fi

# Upgrade gcsfuse to latest version.
if grep -q ubuntu details.txt || grep -q debian details.txt;
then
    sudo apt-get install --only-upgrade gcsfuse |& tee -a ~/logs.txt
else
    sudo yum -y upgrade gcsfuse |& tee -a ~/logs.txt
fi

# Verify that gcsfuse has been upgraded to the to_be_released version using version comparison.
# This is to ensure that the correct version is installed after the upgrade.
gcsfuse --version |& tee version.txt
installed_version=$(echo $(sed -n 1p version.txt) | cut -d' ' -f3)
# The following command compares the two versions:
# 1. `printf` outputs to_release_version and installed_version on a new line.
# 2. `sort -V` sorts them naturally (version sort).
# 3. `tail -n 1` gets the last line, which is the highest version.
# The condition is true if installed_version is greater than or equal to to_release_version.
if [[ "$(printf '%s\n%s\n' "$to_release_version" "$installed_version" | sort -V | tail -n 1)" == "$installed_version" ]]; then
  echo "GCSFuse successfully upgraded to latest version: installed_version ($installed_version), to_release_version: ($to_release_version)" &>> ~/logs.txt
else
  echo "Failure detected in upgrading to latest gcsfuse version: installed_version ($installed_version), to_release_version: ($to_release_version)" &>> ~/logs.txt
fi

if grep -q Failure ~/logs.txt; then
    echo "Test failed" &>> ~/logs.txt ;
else
    touch success.txt
    gsutil cp success.txt gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/installation-test/$(sed -n 3p details.txt)/   ;
fi

gsutil cp ~/logs.txt gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/installation-test/$(sed -n 3p details.txt)/
