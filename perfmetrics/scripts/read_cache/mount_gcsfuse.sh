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

print_usage() {
  echo "Help/Supported options..."
  printf "./mount_gcsfuse.sh  "
  printf "[-s max_size_in_mb] "
  printf "[-b bucket_name] "
  printf "[-c cache_location"
  printf "[-d (means download_for_random_read is true)] \n"
}

# Default value to edit, value via argument will override
# these values if provided.
download_for_random_read=false
max_size_in_mb=100
cache_location=/tmp/read_cache/
bucket_name=princer-read-cache-load-test
stat_cache_ttl=100
type_cache_ttl=100
stat_cache_capacity=4994 # TODO(raj-prince) to update

while getopts hds:b: flag
do
  case "${flag}" in
    s) max_size_in_mb=${OPTARG};;
    b) bucket_name=${OPTARG};;
    d) download_for_random_read=true;;
    h) print_usage
        exit 0 ;;
    *) print_usage
        exit 1 ;;
  esac
done

# Execute the shell from the read_cache scripts directory.
cd $WORKING_DIR/gcsfuse/perfmetrics/scripts/read_cache/

# Create mount dir.
mount_dir=$WORKING_DIR/gcs
mkdir -p $mount_dir

if mountpoint -q -- "$mount_dir"; then
  echo "Unmounting previous mount..."
  umount $mount_dir
fi

# Generate yml config.
export MAX_SIZE_IN_MB="${max_size_in_mb}"
export DOWNLOAD_FOR_RANDOM_READ="${download_for_random_read}"
export CACHE_LOCATION="${cache_location}"
./generate_yml_config.sh

# Mount gcsfuse
echo "Mounting gcsfuse..."
gcsfuse --stackdriver-export-interval 30s \
        --stat-cache-ttl $stat_cache_ttl \
        --stat-cache-capacity $stat_cache_capacity \
        --type-cache-ttl $type_cache_ttl \
        --debug_fuse --debug_gcs \
        --log-file $WORKING_DIR/gcsfuse_logs.txt \
        --log-format text \
        --config-file ./config.yml \
        $bucket_name $mount_dir

# Back to old dir in the last.
cd -