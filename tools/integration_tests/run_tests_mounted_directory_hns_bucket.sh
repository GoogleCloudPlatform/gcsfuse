## Copyright 2024 Google LLC
##
## Licensed under the Apache License, Version 2.0 (the "License");
## you may not use this file except in compliance with the License.
## You may obtain a copy of the License at
##
##	http://www.apache.org/licenses/LICENSE-2.0
##
## Unless required by applicable law or agreed to in writing, software
## distributed under the License is distributed on an "AS IS" BASIS,
## WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
## See the License for the specific language governing permissions and
## limitations under the License.
#
## Run integration tests for mounted directory.
## $1 testbucket.(It should be hierarchical bucket.)
## $2 Absolute path of mounted directory.
## To run this script
## cd gcsfuse
## sh tools/integration_tests/run_tests_mounted_directory_hns_bucket.sh <testbucket> <Absolute path of mounted directory>
#
TEST_BUCKET_NAME=$1
MOUNT_DIR=$2
#export CGO_ENABLED=0
#echo "enable-hns: true" > /tmp/gcsfuse_config.yaml
#
#set -e
## package operations
## Run test with static mounting. (flags: =true)
#go run . --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## Run test with persistent mounting. (flags: =true)
##mount.go run . $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=true,config_file=/tmp/gcsfuse_config.yaml
##GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
##sudo umount $MOUNT_DIR
#
## Run test with static mounting.
#go run . --config-file=/tmp/gcsfuse_config.yaml  $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## Run test with persistent mounting.
##mount.go run . $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=false,config_file=/tmp/gcsfuse_config.yaml
##GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
##sudo umount $MOUNT_DIR
#
## Run test with static mounting. (flags: --experimental-enable-json-read)
#go run . --config-file=/tmp/gcsfuse_config.yaml --experimental-enable-json-read $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## Run test with static mounting. (flags: --kernel-list-cache-ttl-secs=-1, =true)
#go run . --config-file=/tmp/gcsfuse_config.yaml --kernel-list-cache-ttl-secs=-1 $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## Run test with persistent mounting. (flags: --experimental-enable-json-read, =true)
##mount.go run . $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=true,experimental_enable_json_read=true,config_file=/tmp/gcsfuse_config.yaml
##GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
##sudo umount $MOUNT_DIR
#
## Run tests with static mounting. (flags: =true, --only-dir testDir)
#go run . --config-file=/tmp/gcsfuse_config.yaml --only-dir testDir $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
#sudo umount $MOUNT_DIR
#
## Run tests with persistent mounting. (flags: =true, --only-dir=testDir)
##mount.go run . $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs=true,config_file=/tmp/gcsfuse_config.yaml
##GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
##sudo umount $MOUNT_DIR
#
## Run tests with static mounting. (flags: , --only-dir testDir)
#go run . --config-file=/tmp/gcsfuse_config.yaml --only-dir testDir  $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
#sudo umount $MOUNT_DIR
#
## Run tests with persistent mounting. (flags: , --only-dir=testDir)
##mount.go run . $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs=false,config_file=/tmp/gcsfuse_config.yaml
##GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
##sudo umount $MOUNT_DIR
#
## Run tests with only-dir mounting. (flags: --experimental-enable-json-read, --only-dir testDir)
#go run . --config-file=/tmp/gcsfuse_config.yaml --experimental-enable-json-read --only-dir testDir $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
#sudo umount $MOUNT_DIR
#
## Run tests with only-dir mounting. (flags: --kernel-list-cache-ttl-secs=-1, =true, --only-dir testDir)
#go run . --config-file=/tmp/gcsfuse_config.yaml --kernel-list-cache-ttl-secs=-1 --only-dir testDir $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
#sudo umount $MOUNT_DIR
#
## Run tests with config "create-empty-file: true".
#echo "write:
#       create-empty-file: true
#enable-hns: true" > /tmp/gcsfuse_config.yaml
#go run . --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## Run tests with config "file-cache: max-size-mb" static mounting.
#echo "file-cache:
#       max-size-mb: 2
#cache-dir: ./cache-dir
#enable-hns: true" > /tmp/gcsfuse_config.yaml
#go run . --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## Run tests with config "metadata-cache: ttl-secs: 0" static mounting.
#echo "metadata-cache:
#       ttl-secs: 0
#enable-hns: true" > /tmp/gcsfuse_config.yaml
#go run . --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## package readonly
## Run tests with static mounting. (flags: --o=ro)
#go run . --config-file=/tmp/gcsfuse_config.yaml --o=ro $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## Run tests with static mounting. (flags:  --file-mode=544, --dir-mode=544)
#go run . --config-file=/tmp/gcsfuse_config.yaml --file-mode=544 --dir-mode=544  $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## Run tests with static mounting. (flags: --o=ro, --only-dir testDir)
#go run . --config-file=/tmp/gcsfuse_config.yaml --only-dir testDir --o=ro $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
#sudo umount $MOUNT_DIR
#
## Run test with static mounting. (flags: =true, --file-mode=544, --dir-mode=544, --only-dir testDir)
#go run . --config-file=/tmp/gcsfuse_config.yaml --only-dir testDir --file-mode=544 --dir-mode=544  $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
#sudo umount $MOUNT_DIR
#
## Run tests with config "file-cache: max-size-mb" static mounting.
#echo "file-cache:
#       max-size-mb: 3
#cache-dir: ./cache-dir
#enable-hns: true" > /tmp/gcsfuse_config.yaml
#go run . --config-file /tmp/gcsfuse_config.yaml --only-dir testDir --file-mode=544 --dir-mode=544  $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
#sudo umount $MOUNT_DIR
#
## package rename_dir_limit
## Run tests with static mounting.
#go run . --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## Run test with static mounting. (flags: --only-dir testDir)
#go run . --config-file=/tmp/gcsfuse_config.yaml --only-dir testDir $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## package implicit_dir
## Run tests with static mounting.
#go run . --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## package list_large_dir
## Run tests with static mounting.
#go run . --config-file=/tmp/gcsfuse_config.yaml --stat-cache-ttl=0 --kernel-list-cache-ttl-secs=-1 $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/list_large_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## package read_large_files
## Run tests with static mounting.
#go run . $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
#sudo umount $MOUNT_DIR
#
## Run tests with config "file-cache: max-size-mb, cache-file-for-range-read".
#echo "file-cache:
#       max-size-mb: 700
#       cache-file-for-range-read: true
#cache-dir: ./cache-dir
#enable-hns: true
#       " > /tmp/gcsfuse_config.yaml
#go run . --config-file /tmp/gcsfuse_config.yaml  $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
#sudo umount $MOUNT_DIR
#
## Run tests with config "file-cache: max-size-mb".
#echo "file-cache:
#       max-size-mb: -1
#       cache-file-for-range-read: false
#cache-dir: ./cache-dir
#enable-hns: true
#       " > /tmp/gcsfuse_config.yaml
#go run . --config-file /tmp/gcsfuse_config.yaml  $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
#sudo umount $MOUNT_DIR
#
## package write_large_files
## Run tests with static mounting.
#go run . --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/write_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## package gzip
## Run tests with static mounting.
#go run . --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/gzip/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## package local_file
## Run test with static mounting.
#go run . --config-file=/tmp/gcsfuse_config.yaml  $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/local_file/... -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR
#
## Run test with static mounting.
#go run . --config-file=/tmp/gcsfuse_config.yaml   $TEST_BUCKET_NAME $MOUNT_DIR
#GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/local_file/... -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
#sudo umount $MOUNT_DIR

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
      enable-hns: true
       " > /tmp/gcsfuse_config.yaml
go run . --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/log_rotation/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run read cache functional tests.
function read_cache_test_setup() {
    local cache_size_mb=$1
    local enable_range_read_cache=$2
    local cache_ttl=$3
    local enable_parallel_downloads=$4

    cleanup_test_environment
    generate_config_file "$cache_size_mb" "$enable_range_read_cache" "$cache_ttl" "$enable_parallel_downloads"
}

function cleanup_test_environment() {
    # Clean up any pre-existing log files and cache directory.
    rm -rf /tmp/gcsfuse_read_cache_test_logs /tmp/cache-dir
    mkdir -p /tmp/gcsfuse_read_cache_test_logs /tmp/cache-dir
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
enable-hns: true
cache-dir: /tmp/cache-dir" > /tmp/gcsfuse_config.yaml
}

function run_read_cache_test() {
    local test_case=$1
    local optional_flags=$2

    if [ -n "$optional_flags" ]; then
      go run . "$optional_flags" --config-file=/tmp/gcsfuse_config.yaml "$TEST_BUCKET_NAME" "$MOUNT_DIR" > /dev/null
    else
      go run . --config-file=/tmp/gcsfuse_config.yaml "$TEST_BUCKET_NAME" "$MOUNT_DIR" > /dev/null
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

# Read-cache test with cache-file-for-range-read:true.
test_case="TestCacheFileForRangeReadTrueTest/TestRangeReadsWithCacheHit"
# 1. With disabled parallel downloads.
read_cache_test_setup 50 true 3600 false
run_read_cache_test "$test_case"
# 2. With enabled parallel downloads.
read_cache_test_setup 50 true 3600 true
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

# For GRPC: running only core integration tests.

# Test packages: operations
# Run test with static mounting. (flags: --client-protocol=grpc =true)
go run . --config-file=/tmp/gcsfuse_config.yaml --client-protocol=grpc $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Test package: implicit_dir
# Run tests with static mounting.  (flags: --client-protocol=grpc =true)
go run . --config-file=/tmp/gcsfuse_config.yaml --client-protocol=grpc $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Test package: concurrent_operations
# Run tests with static mounting.  (flags: --kernel-list-cache-ttl-secs=-1 =true)
go run . --config-file=/tmp/gcsfuse_config.yaml --kernel-list-cache-ttl-secs=-1 $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/concurrent_operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
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
  "TestInfiniteKernelListCacheTest/TestKernelListCache_ListAndDeleteDirectory"
  "TestInfiniteKernelListCacheTest/TestKernelListCache_DeleteAndListDirectory"
)
for test_case in "${test_cases[@]}"; do
  go run . --config-file=/tmp/gcsfuse_config.yaml --kernel-list-cache-ttl-secs=-1 "$TEST_BUCKET_NAME" "$MOUNT_DIR"
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/kernel-list-cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$test_case"
  sudo umount "$MOUNT_DIR"
done

# Kernel list cache with finite ttl (--kernel-list-cache-ttl-secs=5).
test_cases=(
  "TestFiniteKernelListCacheTest/TestKernelListCache_CacheHitWithinLimit_CacheMissAfterLimit"
)
for test_case in "${test_cases[@]}"; do
  go run . --config-file=/tmp/gcsfuse_config.yaml --kernel-list-cache-ttl-secs=5  "$TEST_BUCKET_NAME" "$MOUNT_DIR"
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/kernel-list-cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$test_case"
  sudo umount "$MOUNT_DIR"
done

# Disabled Kernel list cache (--kernel-list-cache-ttl-secs=0 --stat-cache-ttl=0 ).
test_cases=(
  "TestDisabledKernelListCacheTest/TestKernelListCache_AlwaysCacheMiss"
)
for test_case in "${test_cases[@]}"; do
  go run . --config-file=/tmp/gcsfuse_config.yaml --kernel-list-cache-ttl-secs=0 --stat-cache-ttl=0  "$TEST_BUCKET_NAME" "$MOUNT_DIR"
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/kernel-list-cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$test_case"
  sudo umount "$MOUNT_DIR"
done
