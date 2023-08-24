#! /bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
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
curl http://metadata.google.internal/computeMetadata/v1/instance/name -H "Metadata-Flavor: Google" >> details.txt
touch ~/logs.txt

# Based on the os type(from vm instance name) in detail.txt, run the following commands to install apt-transport-artifact-registry
if grep -q ubuntu details.txt || grep -q debian details.txt;
then
#  For ubuntu and debian os
    curl https://us-central1-apt.pkg.dev/doc/repo-signing-key.gpg | sudo apt-key add - && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
    echo 'deb http://packages.cloud.google.com/apt apt-transport-artifact-registry-stable main' | sudo tee -a /etc/apt/sources.list.d/artifact-registry.list
    sudo apt update
    sudo apt install apt-transport-artifact-registry
    echo "deb ar+https://us-apt.pkg.dev/projects/gcs-fuse-prod gcsfuse-$(lsb_release -cs) main" | sudo tee -a /etc/apt/sources.list.d/artifact-registry.list
    sudo apt update

    # Install released gcsfuse version
    sudo apt install -y gcsfuse=$(sed -n 1p details.txt) -t gcsfuse-$(lsb_release -cs) |& tee -a ~/logs.txt
else
#  For rhel and centos
    sudo yum makecache
    sudo yum -y install yum-plugin-artifact-registry
sudo tee -a /etc/yum.repos.d/artifact-registry.repo << EOF
[gcsfuse-el7-x86-64]
name=gcsfuse-el7-x86-64
baseurl=https://asia-yum.pkg.dev/projects/gcs-fuse-prod/gcsfuse-el7-x86-64
enabled=1
repo_gpgcheck=0
gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
    https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF
    sudo yum makecache
    sudo yum -y --enablerepo=gcsfuse-el7-x86-64 install gcsfuse-$(sed -n 1p details.txt)-1 |& tee -a ~/logs.txt
fi

# Verify gcsfuse version (successful installation)
gcsfuse --version |& tee version.txt
installed_version=$(echo $(sed -n 1p version.txt) | cut -d' ' -f3)
if grep -q $installed_version details.txt; then
    echo "GCSFuse latest version installed correctly." &>> ~/logs.txt
else
    echo "Failure detected in latest gcsfuse version installation." &>> ~/logs.txt
fi

# Uninstall gcsfuse latest version and install old version
if grep -q ubuntu details.txt || grep -q debian details.txt;
then
    sudo apt remove -y gcsfuse
    sudo apt install -y gcsfuse=0.42.5 -t gcsfuse-$(lsb_release -cs) |& tee -a ~/logs.txt
else
    sudo yum -y remove gcsfuse
    sudo yum -y install gcsfuse-0.42.5-1 |& tee -a ~/logs.txt
fi

# verify old version installation
gcsfuse --version |& tee version.txt
installed_version=$(echo $(sed -n 1p version.txt) | cut -d' ' -f3)
if [ $installed_version == "0.42.5" ]; then
    echo "GCSFuse old version (0.42.5) installed successfully" &>> ~/logs.txt
else
    echo "Failure detected in GCSFuse old version installation." &>> ~/logs.txt
fi

# Upgrade gcsfuse to latest version
if grep -q ubuntu details.txt || grep -q debian details.txt;
then
    sudo apt install --only-upgrade gcsfuse |& tee -a ~/logs.txt
else
    sudo yum -y upgrade gcsfuse |& tee -a ~/logs.txt
fi

gcsfuse --version |& tee version.txt
installed_version=$(echo $(sed -n 1p version.txt) | cut -d' ' -f3)
if grep -q $installed_version details.txt; then
    echo "GCSFuse successfully upgraded to latest version $installed_version." &>> ~/logs.txt
else
    echo "Failure detected in upgrading to latest gcsfuse version." &>> ~/logs.txt
fi

if grep -q Failure ~/logs.txt; then
    echo "Test failed" &>> ~/logs.txt ;
else
    touch success.txt
    gsutil cp success.txt gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/installation-test/$(sed -n 3p details.txt)/   ;
fi

gsutil cp ~/logs.txt gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/installation-test/$(sed -n 3p details.txt)/
