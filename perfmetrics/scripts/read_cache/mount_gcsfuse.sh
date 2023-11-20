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

bucket_name=princer-read-cache-load-test

# Create mount dir.
mount_dir=$WORKING_DIR/gcs
mkdir -p $mount_dir

# Mount gcsfuse
gcsfuse --stackdriver-export-interval 30s --debug_fuse --log-file $WORKING_DIR/gcsfuse_logs.txt --log-format text $bucket_name $mount_dir
