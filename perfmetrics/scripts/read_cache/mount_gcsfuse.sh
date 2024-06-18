#!/bin/bash
# Copyright 2023 Google Inc. All Rights Reserved.
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

set -x # Verbose output.
set -e

print_usage() {
  echo "Help/Supported options..."
  printf "./mount_gcsfuse.sh  "
  printf "[-s max_size_mb] "
  printf "[-b bucket_name] "
  printf "[-c cache_dir] "
  printf "[-d (means cache_file_for_range_read is true)] "
  printf "[-m (means metadata cache max size in mb)] "
  printf "[-l (start_logging on $WORKING_DIR/gcsfuse_logs.txt)] \n"
}

# Default value to edit, value via argument will override
# these values if provided.
cache_file_for_range_read=false
max_size_mb=100
cache_dir=/tmp/read_cache/
bucket_name=gcsfuse-read-cache-fio-load-test
stat_or_type_cache_ttl_secs=6048000 # 168h or 1 week
stat_cache_max_size_mb=10
enable_log=0

while getopts lhds:b:c:m: flag
do
  case "${flag}" in

    s) max_size_mb=${OPTARG};;
    b) bucket_name=${OPTARG};;
    d) cache_file_for_range_read=true;;
    c) cache_dir=${OPTARG};;
    m) stat_cache_max_size_mb=${OPTARG};;
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

  # Allowing sufficient time for the completion of the previous unmount process.
  # Why? - Sporadic failures have been observed within the file-cache flow, particularly
  # when the subsequent mount operation immediately follows the unmount operation. By default,
  # the unmount process occurs lazily, during which time the cache folder is deleted.
  # This lazy deletion creates challenges in the subsequent mount process. Specifically,
  # it is possible that the lazy deletion will delete a file after it has been created during
  # the next mount operation. This results in an inconsistency where the fileInfoCache contains
  # an entry for a file that is no longer present on the system.
  sleep 10
fi

# Generate yml config.
export MAX_SIZE_MB="${max_size_mb}"
export CACHE_FILE_FOR_RANGE_READ="${cache_file_for_range_read}"
export CACHE_DIR="${cache_dir}"
export TTL_SECS="${stat_or_type_cache_ttl_secs}"
export STAT_CACHE_MAX_SIZE_MB="${stat_cache_max_size_mb}"
./generate_yml_config.sh

debug_flags=""
if [ $enable_log -eq 1 ]; then
  debug_flags="--debug_fuse --debug_gcs"
fi

# Mount gcsfuse
echo "Mounting gcsfuse..."
gcsfuse --stackdriver-export-interval 30s \
        $debug_flags \
        --log-file $WORKING_DIR/gcsfuse_logs.txt \
        --log-format text \
        --config-file ./config.yml \
        $bucket_name $mount_dir

# Using "gcsfuse --foreground" as grep search query to uniquely identify gcsfuse running process.
gcsfuseID=$(ps -ax | grep "gcsfuse --foreground" | head -n 1 | awk '{print $1}')
echo "Running GCSFuse process-id: $gcsfuseID"

# Back to old dir in the last.
cd -
