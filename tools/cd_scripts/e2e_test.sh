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

echo User: $USER &>> ~/logs.txt
echo Current Working Directory: $(pwd)  &>> ~/logs.txt

# Based on the os type in detail.txt, run the following commands for setup
if grep -q ubuntu details.txt || grep -q debian details.txt;
then
#  For Debian and Ubuntu os
    sudo apt update

    #Install fuse
    sudo apt install -y fuse

    # download and install gcsfuse deb package
    gsutil cp gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/gcsfuse_$(sed -n 1p details.txt)_amd64.deb .
    sudo dpkg -i gcsfuse_$(sed -n 1p details.txt)_amd64.deb |& tee -a ~/logs.txt

    # install wget
    sudo apt install -y wget

    #install git
    sudo apt install -y git

    #install build-essentials
    sudo apt install -y build-essential
else
#  For rhel and centos
    sudo yum makecache
    sudo yum -y update

    #Install fuse
    sudo yum -y install fuse

    #download and install gcsfuse rpm package
    gsutil cp gs://gcsfuse-release-packages/v$(sed -n 1p details.txt)/gcsfuse-$(sed -n 1p details.txt)-1.x86_64.rpm .
    sudo yum -y localinstall gcsfuse-$(sed -n 1p details.txt)-1.x86_64.rpm

    #install wget
    sudo yum -y install wget

    #install git
    sudo yum -y install git

    #install Development tools
    sudo yum -y install gcc gcc-c++ make
fi

# install go
wget -O go_tar.tar.gz https://go.dev/dl/go1.20.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go_tar.tar.gz
export PATH=${PATH}:/usr/local/go/bin

#Write gcsfuse and go version to log file
gcsfuse --version |& tee -a ~/logs.txt
go version |& tee -a ~/logs.txt

# Clone and checkout gcsfuse repo
export PATH=${PATH}:/usr/local/go/bin
git clone https://github.com/googlecloudplatform/gcsfuse |& tee -a ~/logs.txt
cd gcsfuse
git checkout $(sed -n 2p ~/details.txt) |& tee -a ~/logs.txt

#run tests with testbucket flag
set +e
GODEBUG=asyncpreemptoff=1 CGO_ENABLED=0 go test ./tools/integration_tests/... -p 1 --integrationTest -v --testbucket=$(sed -n 3p ~/details.txt) --testInstalledPackage --timeout=60m &>> ~/logs.txt

if [ $? -ne 0 ];
then
    echo "Test failures detected" &>> ~/logs.txt
else
    touch success.txt
    gsutil cp success.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
fi

gsutil cp ~/logs.txt gs://gcsfuse-release-packages/v$(sed -n 1p ~/details.txt)/$(sed -n 3p ~/details.txt)/
'
