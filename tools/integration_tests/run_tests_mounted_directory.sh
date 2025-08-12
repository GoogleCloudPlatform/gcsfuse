# Copyright 2023 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#	http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Run integration tests for mounted directory.
# $1 testbucket.
# $2 Absolute path of mounted directory.
# To run this script
# cd gcsfuse
# sh tools/integration_tests/run_tests_mounted_directory.sh <testbucket> <Absolute path of mounted directory>

TEST_BUCKET_NAME=$1
MOUNT_DIR=$2
export CGO_ENABLED=0

ZONAL_BUCKET_ARG=
if [ $# -gt 2 ] ; then
  if [ "$3" = "true" ]; then
    ZONAL_BUCKET_ARG="--zonal=true"
  else if [ "$3" != "false" ]; then
    >&2 echo "Unexpected value of RUN_ZONAL_BUCKET: $3. Expected: true or false."
    exit 1
  fi
fi

# package operations
# Run test with static mounting. (flags: --implicit-dirs=true)
gcsfuse --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=true)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --implicit-dirs=false)
gcsfuse --implicit-dirs=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=false)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=false
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --experimental-enable-json-read)
gcsfuse --experimental-enable-json-read $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --kernel-list-cache-ttl-secs=-1, --implicit-dirs=true)
gcsfuse --kernel-list-cache-ttl-secs=-1 --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --experimental-enable-json-read, --implicit-dirs=true)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=true,experimental_enable_json_read=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --implicit-dirs=true, --only-dir testDir)
gcsfuse --only-dir testDir --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with persistent mounting. (flags: --implicit-dirs=true, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --implicit-dirs=false, --only-dir testDir)
gcsfuse --only-dir testDir --implicit-dirs=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with persistent mounting. (flags: --implicit-dirs=false, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs=false
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with only-dir mounting. (flags: --experimental-enable-json-read, --only-dir testDir)
gcsfuse --experimental-enable-json-read --only-dir testDir $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with only-dir mounting. (flags: --kernel-list-cache-ttl-secs=-1, --implicit-dirs=true, --only-dir testDir)
gcsfuse --kernel-list-cache-ttl-secs=-1 --implicit-dirs --only-dir testDir $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with persistent mounting. (flags: --experimental-enable-json-read, --implicit-dirs=true, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs=true,experimental_enable_json_read=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with config "create-empty-file: true".
echo "write:
       create-empty-file: true
       " > /tmp/gcsfuse_config.yaml
gcsfuse --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with config "file-cache: max-size-mb" static mounting.
echo "file-cache:
       max-size-mb: 2
cache-dir: /tmp/cache-dir-operations-hns-false
       " > /tmp/gcsfuse_config.yaml
gcsfuse --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with config "metadata-cache: ttl-secs: 0" static mounting.
echo "metadata-cache:
       ttl-secs: 0
       " > /tmp/gcsfuse_config.yaml
gcsfuse --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# package readonly
# Run tests with static mounting. (flags: --implicit-dirs=true,--o=ro)
gcsfuse --o=ro --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=true,--o=ro)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o ro,implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --implicit-dirs=true, --file-mode=544, --dir-mode=544)
gcsfuse --file-mode=544 --dir-mode=544 --implicit-dirs=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=true, --file-mode=544, --dir-mode=544)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o file_mode=544,dir_mode=544,implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --implicit-dirs=true, --o=ro, --only-dir testDir)
gcsfuse --only-dir testDir --o=ro --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting.  (flags: --implicit-dirs=true,--o=ro,--only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o ro,only_dir=testDir,implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --implicit-dirs=true, --file-mode=544, --dir-mode=544, --only-dir testDir)
gcsfuse --only-dir testDir --file-mode=544 --dir-mode=544 --implicit-dirs=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=true, --file-mode=544, --dir-mode=544, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,file_mode=544,dir_mode=544,implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with config "file-cache: max-size-mb" static mounting.
echo "file-cache:
       max-size-mb: 3
cache-dir: /tmp/cache-dir-readonly-hns-false
       " > /tmp/gcsfuse_config.yaml
gcsfuse --config-file /tmp/gcsfuse_config.yaml --only-dir testDir --file-mode=544 --dir-mode=544 --implicit-dirs=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# package rename_dir_limit
# Run tests with static mounting. (flags: --rename-dir-limit=3, --implicit-dirs)
gcsfuse --rename-dir-limit=3 --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --rename-dir-limit=3, --implicit-dirs)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o rename_dir_limit=3,implicit_dirs
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --rename-dir-limit=3)
gcsfuse --rename-dir-limit=3  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --rename-dir-limit=3)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o rename_dir_limit=3
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --rename-dir-limit=3, --implicit-dirs, --only-dir testDir)
gcsfuse --only-dir testDir --rename-dir-limit=3 --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting . (flags: --rename-dir-limit=3, --implicit-dirs)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,rename_dir_limit=3,implicit_dirs
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --rename-dir-limit=3, --only-dir testDir)
gcsfuse --only-dir testDir --rename-dir-limit=3  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting . (flags: --rename-dir-limit=3, --implicit-dirs, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,rename_dir_limit=3,implicit_dirs
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# package implicit_dir
# Run tests with static mounting. (flags: --implicit-dirs)
gcsfuse --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --implicit-dirs, --only-dir testDir)
gcsfuse --only-dir testDir --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs,--only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# package explicit_dir
# Run tests with static mounting. (flags: --implicit-dirs=false)
gcsfuse --implicit-dirs=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/explicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=false)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=false
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/explicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --implicit-dirs=false, --only-dir testDir)
gcsfuse --only-dir testDir  --implicit-dirs=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/explicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=false, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs=false
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/explicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# package list_large_dir
# Run tests with static mounting. (flags: --implicit-dirs)
gcsfuse --implicit-dirs --stat-cache-ttl=0 --kernel-list-cache-ttl-secs=-1 $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/list_large_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# package read_large_files
# Run tests with static mounting. (flags: --implicit-dirs)
gcsfuse --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with config "file-cache: max-size-mb, cache-file-for-range-read".
echo "file-cache:
       max-size-mb: 700
       cache-file-for-range-read: true
cache-dir: /tmp/cache-dir-read-large-files-hns-false
       " > /tmp/gcsfuse_config.yaml
gcsfuse --config-file /tmp/gcsfuse_config.yaml --implicit-dirs=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with config "file-cache: max-size-mb".
echo "file-cache:
       max-size-mb: -1
       cache-file-for-range-read: false
cache-dir: /tmp/cache-dir-read-large-files-hns-false
       " > /tmp/gcsfuse_config.yaml
gcsfuse --config-file /tmp/gcsfuse_config.yaml --implicit-dirs=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# package write_large_files
# Run tests with static mounting. (flags: --implicit-dirs)
gcsfuse --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/write_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# package gzip
# Run tests with static mounting. (flags: --implicit-dirs)
gcsfuse --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/gzip/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# package local_file
# Run test with static mounting. (flags: --implicit-dirs=true)
gcsfuse --implicit-dirs=true --rename-dir-limit=3 $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/local_file/... -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --implicit-dirs=false)
gcsfuse --implicit-dirs=false --rename-dir-limit=3 $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/local_file/... -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run tests with log rotation config.
rm -r /tmp/gcsfuse_integration_test_logs
mkdir /tmp/gcsfuse_integration_test_logs
echo "logging:
        file-path: /tmp/gcsfuse_integration_test_logs/log.txt
        format: text
        severity: trace
        log-rotate:
          max-file-size-mb: 2
          backup-file-count: 3
          compress: true
       " > /tmp/gcsfuse_config.yaml
gcsfuse --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/log_rotation/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run read cache functional tests.
function read_cache_test_setup() {
    local cache_size_mb=$1
    local enable_range_read_cache=$2
    local cache_ttl=$3
    local enable_parallel_downloads=$4
    local enable_o_direct=$5
    if [ -n "$enable_o_direct" ]; then
      enable_o_direct=false
    else
      enable_o_direct=true
    fi

    cleanup_test_environment
    generate_config_file "$cache_size_mb" "$enable_range_read_cache" "$cache_ttl" "$enable_parallel_downloads" "$enable_o_direct"
}

function cleanup_test_environment() {
    # Clean up any pre-existing log files and cache directory.
    rm -rf /tmp/gcsfuse_read_cache_test_logs /tmp/cache-dir-read-cache-hns-false
    mkdir -p /tmp/gcsfuse_read_cache_test_logs /tmp/cache-dir-read-cache-hns-false
}

function generate_config_file() {
  local cache_size_mb=$1
  local enable_range_read_cache=$2
  local cache_ttl=$3
  local enable_parallel_downloads=$4

  echo "logging:
  file-path: /tmp/gcsfuse_read_cache_test_logs/log.json
  format: json
  severity: trace
file-cache:
  max-size-mb: $cache_size_mb
  cache-file-for-range-read: $enable_range_read_cache
  enable-parallel-downloads: $enable_parallel_downloads
  parallel-downloads-per-file: 4
  max-parallel-downloads: -1
  download-chunk-size-mb: 3
  enable-crc: true
metadata-cache:
  stat-cache-max-size-mb: 4
  ttl-secs: $cache_ttl
  type-cache-max-size-mb: 32
cache-dir: /tmp/cache-dir-read-cache-hns-false" > /tmp/gcsfuse_config.yaml
}

function run_read_cache_test() {
    local test_case=$1
    local optional_flags=$2

    if [ -n "$optional_flags" ]; then
      gcsfuse "$optional_flags" --config-file=/tmp/gcsfuse_config.yaml "$TEST_BUCKET_NAME" "$MOUNT_DIR" > /dev/null
    else
      gcsfuse --config-file=/tmp/gcsfuse_config.yaml "$TEST_BUCKET_NAME" "$MOUNT_DIR" > /dev/null
    fi
    GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$test_case"
    sudo umount "$MOUNT_DIR"
    cleanup_test_environment
}

# Read-cache test with cache-file-for-range-read:false.
test_cases=(
  "TestCacheFileForRangeReadFalseTest/TestRangeReadsWithCacheMiss"
  "TestCacheFileForRangeReadFalseTest/TestConcurrentReads_ReadIsTreatedNonSequentialAfterFileIsRemovedFromCache"
)
# 1. With disabled parallel downloads.
read_cache_test_setup 50 false 3600 false
for test_case in "${test_cases[@]}"; do
  run_read_cache_test "$test_case"
done
# 2. With enabled parallel downloads.
read_cache_test_setup 50 false 3600 true
run_read_cache_test "${test_cases[0]}"
# 3. With enabled parallel downloads and enabled O_DIRECT
read_cache_test_setup 50 false 3600 true true
run_read_cache_test "${test_cases[0]}"

# Read-cache test with cache-file-for-range-read:true.
test_case="TestCacheFileForRangeReadTrueTest/TestRangeReadsWithCacheHit"
# 1. With disabled parallel downloads.
read_cache_test_setup 50 true 3600 false
run_read_cache_test "$test_case"
# 2. With enabled parallel downloads.
read_cache_test_setup 50 true 3600 true
run_read_cache_test "$test_case"
# 3. With enabled parallel downloads and enabled O_DIRECT
read_cache_test_setup 50 true 3600 true true
run_read_cache_test "$test_case"

# Read-cache test with disabled cache ttl.
test_case="TestDisabledCacheTTLTest/TestReadAfterObjectUpdateIsCacheMiss"
# 1. With disabled parallel downloads.
read_cache_test_setup 9 false 0 false
run_read_cache_test "$test_case"
# 2. With enabled parallel downloads.
read_cache_test_setup 9 false 0 true
run_read_cache_test "$test_case"

# Read-cache test with local modification.
test_case="TestLocalModificationTest/TestReadAfterLocalGCSFuseWriteIsCacheMiss"
# 1. With disabled parallel downloads.
read_cache_test_setup 9 false 3600 false
run_read_cache_test "$test_case"
# 2. With enabled parallel downloads.
read_cache_test_setup 9 false 3600 true
run_read_cache_test "$test_case"

# Read-cache tests for range reads.
test_cases=(
  "TestRangeReadTest/TestRangeReadsWithinReadChunkSize"
  "TestRangeReadTest/TestRangeReadsBeyondReadChunkSizeWithChunkDownloaded"
)
for test_case in "${test_cases[@]}"; do
  # 1. With disabled parallel downloads.
  read_cache_test_setup 500 false 3600 false
  run_read_cache_test "$test_case"
  read_cache_test_setup 500 true 3600 false
  run_read_cache_test "$test_case"
done
# 2. With enabled parallel downloads.
read_cache_test_setup 500 false 3600 true
run_read_cache_test "${test_cases[1]}"
read_cache_test_setup 500 true 3600 true
run_read_cache_test "${test_cases[1]}"

# Read cache tests on read only mount.
test_cases=(
  "TestReadOnlyTest/TestSecondSequentialReadIsCacheHit"
  "TestReadOnlyTest/TestReadFileSequentiallyLargerThanCacheCapacity"
  "TestReadOnlyTest/TestReadFileRandomlyLargerThanCacheCapacity"
  "TestReadOnlyTest/TestReadMultipleFilesMoreThanCacheLimit"
  "TestReadOnlyTest/TestReadMultipleFilesWithinCacheLimit"
)

for test_case in "${test_cases[@]}"; do
  # 1. With disabled parallel downloads.
  read_cache_test_setup 9 false 3600 false
  run_read_cache_test "$test_case" "--o=ro"
  read_cache_test_setup 9 true 3600 false
  run_read_cache_test "$test_case" "--o=ro"
  # 2. With enabled parallel downloads.
  read_cache_test_setup 9 false 3600 true
  run_read_cache_test "$test_case" "--o=ro"
  read_cache_test_setup 9 true 3600 true
  run_read_cache_test "$test_case" "--o=ro"
done

# Read cache tests with small cache ttl.
test_cases=(
  "TestSmallCacheTTLTest/TestReadAfterUpdateAndCacheExpiryIsCacheMiss"
  "TestSmallCacheTTLTest/TestReadForLowMetaDataCacheTTLIsCacheHit"
)
for test_case in "${test_cases[@]}"; do
  # 1. With disabled parallel downloads.
  read_cache_test_setup 9 false 10 false
  run_read_cache_test "$test_case"
  # 2. With enabled parallel downloads.
  read_cache_test_setup 9 false 10 true
  run_read_cache_test "$test_case"
done

# Package managed_folders
echo "list:
  enable-empty-managed-folders: true" > /tmp/gcsfuse_config.yaml
# Empty managed folders listing test.
# Run test with static mounting (flags: --implicit-dirs)
gcsfuse --implicit-dirs --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/managed_folders/... -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR  --testbucket=$TEST_BUCKET_NAME -run TestEnableEmptyManagedFoldersTrue ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs -o config_file=/tmp/gcsfuse_config.yaml
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/managed_folders/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME -run TestEnableEmptyManagedFoldersTrue ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# For GRPC: running only core integration tests.

# Test packages: operations
# Run test with static mounting. (flags: --client-protocol=grpc --implicit-dirs=true)
gcsfuse --client-protocol=grpc --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --client-protocol=grpc --implicit-dirs=true)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=true,client_protocol=grpc
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Test package: implicit_dir
# Run tests with static mounting.  (flags: --client-protocol=grpc --implicit-dirs=true)
gcsfuse --implicit-dirs=true --client-protocol=grpc $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting.  (flags: --client-protocol=grpc --implicit-dirs=true)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=true,client_protocol=grpc
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Test package: concurrent_operations
# Run tests with static mounting.  (flags: --kernel-list-cache-ttl-secs=-1 --implicit-dirs=true)
gcsfuse --implicit-dirs=true --kernel-list-cache-ttl-secs=-1 $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/concurrent_operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting.  (flags: --kernel-list-cache-ttl-secs=-1 --implicit-dirs=true)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=true,kernel_list_cache_ttl_secs=-1
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/concurrent_operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Test package: benchmarking
# Run tests with static mounting.  (flags: --implicit-dirs=true)
gcsfuse --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/benchmarking/... --bench=. -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting.  (flags: --implicit-dirs=true)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/benchmarking/... --bench=. -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Test package: kernel-list-cache

# Kernel list cache with infinite ttl. (--kernel-list-cache-ttl-secs=-1)
test_cases=(
  "TestInfiniteKernelListCacheTest/TestKernelListCache_AlwaysCacheHit"
  "TestInfiniteKernelListCacheTest/TestKernelListCache_CacheMissOnAdditionOfFile"
  "TestInfiniteKernelListCacheTest/TestKernelListCache_CacheMissOnDeletionOfFile"
  "TestInfiniteKernelListCacheTest/TestKernelListCache_CacheMissOnFileRename"
  "TestInfiniteKernelListCacheTest/TestKernelListCache_EvictCacheEntryOfOnlyDirectParent"
  "TestInfiniteKernelListCacheTest/TestKernelListCache_CacheMissOnAdditionOfDirectory"
  "TestInfiniteKernelListCacheTest/TestKernelListCache_CacheMissOnDeletionOfDirectory"
  "TestInfiniteKernelListCacheTest/TestKernelListCache_CacheMissOnDirectoryRename"
)
for test_case in "${test_cases[@]}"; do
  gcsfuse --kernel-list-cache-ttl-secs=-1  "$TEST_BUCKET_NAME" "$MOUNT_DIR"
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/kernel_list_cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$test_case"
  sudo umount "$MOUNT_DIR"
done

test_cases=(
  "TestInfiniteKernelListCacheDeleteDirTest/TestKernelListCache_ListAndDeleteDirectory"
  "TestInfiniteKernelListCacheDeleteDirTest/TestKernelListCache_DeleteAndListDirectory"
)
for test_case in "${test_cases[@]}"; do
  gcsfuse --kernel-list-cache-ttl-secs=-1 --metadata-cache-ttl-secs=0 "$TEST_BUCKET_NAME" "$MOUNT_DIR"
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/kernel_list_cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$test_case"
  sudo umount "$MOUNT_DIR"
done

# Kernel list cache with finite ttl (--kernel-list-cache-ttl-secs=5).
test_cases=(
  "TestFiniteKernelListCacheTest/TestKernelListCache_CacheHitWithinLimit_CacheMissAfterLimit"
)
for test_case in "${test_cases[@]}"; do
  gcsfuse --kernel-list-cache-ttl-secs=5 --rename-dir-limit=10 "$TEST_BUCKET_NAME" "$MOUNT_DIR"
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/kernel_list_cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$test_case"
  sudo umount "$MOUNT_DIR"
done

# Disabled Kernel list cache (--kernel-list-cache-ttl-secs=0 --stat-cache-ttl=0 --rename-dir-limit=10).
test_cases=(
  "TestDisabledKernelListCacheTest/TestKernelListCache_AlwaysCacheMiss"
)
for test_case in "${test_cases[@]}"; do
  gcsfuse --kernel-list-cache-ttl-secs=0 --stat-cache-ttl=0 --rename-dir-limit=10 "$TEST_BUCKET_NAME" "$MOUNT_DIR"
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/kernel_list_cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$test_case"
  sudo umount "$MOUNT_DIR"
done

# Test package: stale_handle
# Run tests with static mounting.  (flags: --metadata-cache-ttl-secs=0 --precondition-errors=true)
gcsfuse --metadata-cache-ttl-secs=0 --precondition-errors=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/stale_handle/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run test with persistent mounting.  (flags: --metadata-cache-ttl-secs=0 --precondition-errors=true)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o metadata_cache_ttl_secs=0,precondition_errors=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/stale_handle/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Test package: streaming_writes
# Run streaming_writes tests.
gcsfuse --rename-dir-limit=3 --enable-streaming-writes=true --write-block-size-mb=1 --write-max-blocks-per-file=2 $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/streaming_writes/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Run write_large_files tests with streaming writes enabled.
gcsfuse --enable-streaming-writes=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/write_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME ${ZONAL_BUCKET_ARG}
sudo umount $MOUNT_DIR

# Rename symlink tests.
# Due to a known bug these tests should not run with gRPC.
gcsfuse --implicit-dirs --metadata-cache-negative-ttl-secs=0 $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_symlink/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR