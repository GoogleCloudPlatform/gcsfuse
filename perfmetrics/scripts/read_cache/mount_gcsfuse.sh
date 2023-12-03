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

print_usage() {
  echo "Help/Supported options..."
  printf "./mount_gcsfuse.sh  "
  printf "[-s max_size_in_mb] "
  printf "[-b bucket_name] "
  printf "[-c cache_location] "
  printf "[-d (means download_for_random_read is true)] "
  printf "[-l (start_logging on $WORKING_DIR/gcsfuse_logs.txt)] \n"
}

# Default value to edit, value via argument will override
# these values if provided.
download_for_random_read=false
max_size_in_mb=100
cache_location=/tmp/read_cache/
bucket_name=gcsfuse-read-cache-fio-load-test
stat_cache_ttl=168h # 1 week
type_cache_ttl=168h # 1 week
stat_cache_capacity=1200000 # 1 million + buffer 200k
enable_log=0

while getopts lhds:b:c: flag
do
  case "${flag}" in

    s) max_size_in_mb=${OPTARG};;
    b) bucket_name=${OPTARG};;
    d) download_for_random_read=true;;
    c) cache_location=${OPTARG};;
    h) print_usage
        exit 0 ;;
    l) enable_log=1 ;;
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

debug_flags=""
if [ $enable_log -eq 1 ]; then
  debug_flags="--debug_fuse --debug_gcs"
fi

# Mount gcsfuse
echo "Mounting gcsfuse..."
gcsfuse --stackdriver-export-interval 30s \
        --stat-cache-ttl $stat_cache_ttl \
        --stat-cache-capacity $stat_cache_capacity \
        --type-cache-ttl $type_cache_ttl \
        $debug_flags \
        --log-file $WORKING_DIR/gcsfuse_logs.txt \
        --log-format text \
        --config-file ./config.yml \
        $bucket_name $mount_dir


gcsfuseID=$(ps -ax | grep "gcsfuse" | head -n 1 | awk '{print $1}')
echo "Running GCSFuse process-id: $gcsfuseID"

# Back to old dir in the last.
cd -
