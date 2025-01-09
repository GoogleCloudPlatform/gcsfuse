#!/bin/bash
# Copyright 2024 Google LLC
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

set -e
# Refer for env var: https://stackoverflow.com/questions/72441758/typeerror-descriptors-cannot-not-be-created-directly
export PROTOCOL_BUFFERS_PYTHON_IMPLEMENTATION=python
echo Installing pip and fuse..
sudo apt-get install fuse -y
sudo apt-get install pip -y
echo Installing requirements..
pip install --require-hashes -r requirements.txt --user
echo Running script..
GCSFUSE_FLAGS=$1
UPLOAD_FLAGS=$2
SPREADSHEET_ID=$3
CONFIG_FILE=$4
python3 listing_benchmark.py $CONFIG_FILE --gcsfuse_flags "$GCSFUSE_FLAGS" $UPLOAD_FLAGS --command "ls -R" --num_samples 30 --message "Testing CT setup." --spreadsheet_id=$SPREADSHEET_ID
