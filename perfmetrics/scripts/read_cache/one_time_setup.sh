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

set -e

if [ ! -d "/mnt/disks/local_ssd" ]; then
  # Mount local_ssd at path /mnt/disks/local_ssd.
  sudo mdadm --create /dev/md0 --level=0 --raid-devices=16 \
    /dev/disk/by-id/google-local-nvme-ssd-0 \
    /dev/disk/by-id/google-local-nvme-ssd-1 \
    /dev/disk/by-id/google-local-nvme-ssd-2 \
    /dev/disk/by-id/google-local-nvme-ssd-3 \
    /dev/disk/by-id/google-local-nvme-ssd-4 \
    /dev/disk/by-id/google-local-nvme-ssd-5 \
    /dev/disk/by-id/google-local-nvme-ssd-6 \
    /dev/disk/by-id/google-local-nvme-ssd-7 \
    /dev/disk/by-id/google-local-nvme-ssd-8 \
    /dev/disk/by-id/google-local-nvme-ssd-9 \
    /dev/disk/by-id/google-local-nvme-ssd-10 \
    /dev/disk/by-id/google-local-nvme-ssd-11 \
    /dev/disk/by-id/google-local-nvme-ssd-12 \
    /dev/disk/by-id/google-local-nvme-ssd-13 \
    /dev/disk/by-id/google-local-nvme-ssd-14 \
    /dev/disk/by-id/google-local-nvme-ssd-15

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

# Install fio.
sudo apt update
sudo apt install fio -y

# Install and validate go.
version=1.21.1
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

# Install clone gcsfuse.
if [ ! -d "./gcsfuse" ]; then
  git clone -b  fio_load_test_script https://github.com/GoogleCloudPlatform/gcsfuse.git
fi

# Mount gcsfuse.
$WORKING_DIR/gcsfuse/perfmetrics/scripts/read_cache/mount_gcsfuse.sh

# Add some alias shortcut in ~/.bashrc file.
cat >> ~/.bashrc << EOF
alias mount_gcsfuse=$WORKING_DIR/gcsfuse/perfmetrics/scripts/read_cache/mount_gcsfuse.sh
alias run_load_test=$WORKING_DIR/gcsfuse/perfmetrics/scripts/read_cache/run_load_test.sh
alias run_individual_test=$WORKING_DIR/gcsfuse/perfmetrics/scripts/read_cache/run_read_cache_fio_workload.sh
EOF


# Back to running directory.
cd -
