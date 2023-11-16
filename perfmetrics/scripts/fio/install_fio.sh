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

echo "Installing fio ..."
# install libaio as fio has a dependency on libaio
sudo apt-get install -y libaio-dev

# We are building fio from source because of issue: https://github.com/axboe/fio/issues/1640.
# The fix is not currently released in a package as of 20th Oct, 2023.
# TODO: install fio via package when release > 3.35 is available.
FIO_SRC_DIR="${KOKORO_ARTIFACTS_DIR}/github/fio"
sudo rm -rf "${FIO_SRC_DIR}" && \
git clone https://github.com/axboe/fio.git "$FIO_SRC_DIR"
cd  "$FIO_SRC_DIR" && \
git checkout c5d8ce3fc736210ded83b126c71e3225c7ffd7c9 && \
./configure && make && sudo make install

# Now, print the installed fio version for verification
echo 'fio version='$(fio -version)