#!/bin/bash
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

set -x # Verbose output.
set -e

if [ ! -d "/mnt/disks/local_ssd" ]; then
  # Mount local_ssd at path /mnt/disks/local_ssd.
  sudo mdadm --create /dev/md0 --level=0 --raid-devices=4 \
    /dev/disk/by-id/google-local-nvme-ssd-0 \
    /dev/disk/by-id/google-local-nvme-ssd-1 \
    /dev/disk/by-id/google-local-nvme-ssd-2 \
    /dev/disk/by-id/google-local-nvme-ssd-3

  sudo mdadm --detail --prefer=by-id /dev/md0
  sudo mkfs.ext4 -F /dev/md0
  sudo mkdir -p /mnt/disks/local_ssd
  sudo mount /dev/md0 /mnt/disks/local_ssd
  sudo chmod a+w /mnt/disks/local_ssd
fi

# Check the mounting working.
df -H

# Don't pollute home, create a working dir.
WD=$HOME/working_dir
mkdir -p $WD
cd $WD


echo "Installing fio ..."
FIO_SRC_DIR=$WD/github
mkdir -p $FIO_SRC_DIR

# install libaio as fio has a dependency on libaio
sudo apt-get install -y libaio-dev
sudo apt-get install -y gcc make

# We are building fio from source because of the issue: https://github.com/axboe/fio/issues/1668.
# The sed command below is to address internal bug#309563824.
# As recorded in this bug, fio by-default supports
# clat percentile values to be calculated accurately upto only
# 2^(FIO_IO_U_PLAT_GROUP_NR + 5) ns = 17.17 seconds.
# (with default value of FIO_IO_U_PLAT_GROUP_NR = 29). This change increases it upto 32, to allow
# latencies upto 137.44s to be calculated accurately.
sudo rm -rf "$FIO_SRC_DIR" && \
git clone https://github.com/axboe/fio.git "$FIO_SRC_DIR" && \
cd  "$FIO_SRC_DIR" && \
git checkout fio-3.36 && \
sed -i 's/define \+FIO_IO_U_PLAT_GROUP_NR \+\([0-9]\+\)/define FIO_IO_U_PLAT_GROUP_NR 32/g' stat.h && \
./configure && make && sudo make install
cd -

# Install and validate go.
version=1.22.0
wget -O go_tar.tar.gz https://go.dev/dl/go${version}.linux-amd64.tar.gz -q
sudo rm -rf /usr/local/go
tar -xzf go_tar.tar.gz && sudo mv go /usr/local
export PATH=$PATH:/usr/local/go/bin && go version && rm go_tar.tar.gz

# Add go in the path permanently, so that $HOME/go/bin is visible.
export PATH=$PATH:$HOME/go/bin/
echo 'export PATH=$PATH:$HOME/go/bin/:/usr/local/go/bin' >> ~/.bashrc

# Export WORKING_DIR env variable and add it to ~/.bashrc.
export WORKING_DIR=$WD
echo "export WORKING_DIR=$WD" >> ~/.bashrc

# Install gcsfuse.
CGO_ENABLED=0 go install github.com/googlecloudplatform/gcsfuse@read_cache_release

# Clone gcsfuse to get fio load test script.
if [ ! -d "./gcsfuse" ]; then
  git clone -b  read_cache_release https://github.com/GoogleCloudPlatform/gcsfuse.git
fi

# Mount gcsfuse.
$WORKING_DIR/gcsfuse/perfmetrics/scripts/read_cache/mount_gcsfuse.sh

# Add some alias shortcut in ~/.bashrc file.
cat >> ~/.bashrc << EOF
alias mount_gcsfuse=$WORKING_DIR/gcsfuse/perfmetrics/scripts/read_cache/mount_gcsfuse.sh
EOF

# Back to running directory.
cd -
