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
go install github.com/googlecloudplatform/gcsfuse@read_cache_release

# Install clone gcsfuse.
if [ ! -d "./gcsfuse" ]; then
  git clone -b  fio_load_test_script https://github.com/GoogleCloudPlatform/gcsfuse.git
fi

# Mount gcsfuse.
$WORKING_DIR/gcsfuse/perfmetrics/scripts/read_cache/mount_gcsfuse.sh

# Back to running directory.
cd -
