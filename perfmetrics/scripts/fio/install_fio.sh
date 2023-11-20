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

# Args: 
# $1: <path-to-clone-fio-source-code> e.g. ~/github/

set -e

if [[ $# -eq 0 ]] ; then
    echo 'No argument passed.'
    echo 'Args: <path-to-clone-fio-source-code>'
    exit 1
fi

SRC_DIR="$1"
FIO_SRC_DIR="${SRC_DIR}/fio"

echo "Installing fio ..."
# install libaio as fio has a dependency on libaio
sudo apt-get install -y libaio-dev

# We are building fio from source because of the issue: https://github.com/axboe/fio/issues/1668.
sudo rm -rf "$FIO_SRC_DIR" && \
git clone https://github.com/axboe/fio.git "$FIO_SRC_DIR" && \
cd  "$FIO_SRC_DIR" && \
git checkout fio-3.36 && \
./configure && make && sudo make install

# Now, print the installed fio version for verification
echo 'fio version='$(fio -version)

# go back to the original directory
cd -
