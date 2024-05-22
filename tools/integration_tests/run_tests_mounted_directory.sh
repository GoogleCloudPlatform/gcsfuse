# Copyright 2023 Google Inc. All Rights Reserved.
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

# package operations
# Run test with static mounting. (flags: --implicit-dirs=true)
gcsfuse --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=true)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --implicit-dirs=false)
gcsfuse --implicit-dirs=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=false)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=false
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --experimental-enable-json-read --implicit-dirs=true)
gcsfuse --experimental-enable-json-read --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --experimental-enable-json-read, --implicit-dirs=true)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=true,experimental_enable_json_read=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --implicit-dirs=true, --only-dir testDir)
gcsfuse --only-dir testDir --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run tests with persistent mounting. (flags: --implicit-dirs=true, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --implicit-dirs=false, --only-dir testDir)
gcsfuse --only-dir testDir --implicit-dirs=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run tests with persistent mounting. (flags: --implicit-dirs=false, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs=false
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --experimental-enable-json-read, --implicit-dirs=true, --only-dir testDir)
gcsfuse --experimental-enable-json-read --only-dir testDir --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run tests with persistent mounting. (flags: --experimental-enable-json-read, --implicit-dirs=true, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs=true,experimental_enable_json_read=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run tests with config "create-empty-file: true".
echo "write:
       create-empty-file: true
       " > /tmp/gcsfuse_config.yaml
gcsfuse --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run tests with config "file-cache: max-size-mb" static mounting.
echo "file-cache:
       max-size-mb: 2
cache-dir: ./cache-dir
       " > /tmp/gcsfuse_config.yaml
gcsfuse --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run tests with config "metadata-cache: ttl-secs: 0" static mounting.
echo "metadata-cache:
       ttl-secs: 0
       " > /tmp/gcsfuse_config.yaml
gcsfuse --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# package readonly
# Run tests with static mounting. (flags: --implicit-dirs=true,--o=ro)
gcsfuse --o=ro --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=true,--o=ro)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o ro,implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --implicit-dirs=true, --file-mode=544, --dir-mode=544)
gcsfuse --file-mode=544 --dir-mode=544 --implicit-dirs=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=true, --file-mode=544, --dir-mode=544)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o file_mode=544,dir_mode=544,implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --implicit-dirs=true, --o=ro, --only-dir testDir)
gcsfuse --only-dir testDir --o=ro --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run test with persistent mounting.  (flags: --implicit-dirs=true,--o=ro,--only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o ro,only_dir=testDir,implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --implicit-dirs=true, --file-mode=544, --dir-mode=544, --only-dir testDir)
gcsfuse --only-dir testDir --file-mode=544 --dir-mode=544 --implicit-dirs=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=true, --file-mode=544, --dir-mode=544, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,file_mode=544,dir_mode=544,implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run tests with config "file-cache: max-size-mb" static mounting.
echo "file-cache:
       max-size-mb: 3
cache-dir: ./cache-dir
       " > /tmp/gcsfuse_config.yaml
gcsfuse --config-file /tmp/gcsfuse_config.yaml --only-dir testDir --file-mode=544 --dir-mode=544 --implicit-dirs=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# package rename_dir_limit
# Run tests with static mounting. (flags: --rename-dir-limit=3, --implicit-dirs)
gcsfuse --rename-dir-limit=3 --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --rename-dir-limit=3, --implicit-dirs)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o rename_dir_limit=3,implicit_dirs
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --rename-dir-limit=3)
gcsfuse --rename-dir-limit=3  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --rename-dir-limit=3)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o rename_dir_limit=3
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --rename-dir-limit=3, --implicit-dirs, --only-dir testDir)
gcsfuse --only-dir testDir --rename-dir-limit=3 --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run test with persistent mounting . (flags: --rename-dir-limit=3, --implicit-dirs)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,rename_dir_limit=3,implicit_dirs
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --rename-dir-limit=3, --only-dir testDir)
gcsfuse --only-dir testDir --rename-dir-limit=3  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run test with persistent mounting . (flags: --rename-dir-limit=3, --implicit-dirs, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,rename_dir_limit=3,implicit_dirs
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# package implicit_dir
# Run tests with static mounting. (flags: --implicit-dirs)
gcsfuse --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --implicit-dirs, --only-dir testDir)
gcsfuse --only-dir testDir --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs,--only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# package explicit_dir
# Run tests with static mounting. (flags: --implicit-dirs=false)
gcsfuse --implicit-dirs=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/explicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=false)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=false
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/explicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run tests with static mounting. (flags: --implicit-dirs=false, --only-dir testDir)
gcsfuse --only-dir testDir  --implicit-dirs=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/explicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=false, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs=false
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/explicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# package list_large_dir
# Run tests with static mounting. (flags: --implicit-dirs)
gcsfuse --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/list_large_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# package read_large_files
# Run tests with static mounting. (flags: --implicit-dirs)
gcsfuse --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run tests with config "file-cache: max-size-mb, cache-file-for-range-read".
echo "file-cache:
       max-size-mb: 700
       cache-file-for-range-read: true
cache-dir: ./cache-dir
       " > /tmp/gcsfuse_config.yaml
gcsfuse --config-file /tmp/gcsfuse_config.yaml --implicit-dirs=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run tests with config "file-cache: max-size-mb".
echo "file-cache:
       max-size-mb: -1
       cache-file-for-range-read: false
cache-dir: ./cache-dir
       " > /tmp/gcsfuse_config.yaml
gcsfuse --config-file /tmp/gcsfuse_config.yaml --implicit-dirs=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# package write_large_files
# Run tests with static mounting. (flags: --implicit-dirs)
gcsfuse --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/write_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# package gzip
# Run tests with static mounting. (flags: --implicit-dirs)
gcsfuse --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/gzip/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# package local_file
# Run test with static mounting. (flags: --implicit-dirs=true)
gcsfuse --implicit-dirs=true --rename-dir-limit=3 $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/local_file/... -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --implicit-dirs=false)
gcsfuse --implicit-dirs=false --rename-dir-limit=3 $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/local_file/... -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
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
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/log_rotation/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run read cache functional tests.
function read_cache_test_setup() {
  cache_size_mb=$1
  enable_range_read_cache=$2
  cache_ttl=$3

  cleanup_test_environment
  generate_config_file "$cache_size_mb" "$enable_range_read_cache" "$cache_ttl"
}

function cleanup_test_environment() {
  # Clean up any pre-existing log files.
  rm -r /tmp/gcsfuse_read_cache_test_logs
  mkdir /tmp/gcsfuse_read_cache_test_logs
  # Clean up cache directory.
  rm -r /tmp/cache-dir
  mkdir /tmp/cache-dir
}

function generate_config_file() {
  cache_size_mb=$1
  enable_range_read_cache=$2
  cache_ttl=$3

  echo "logging:
  file-path: /tmp/gcsfuse_read_cache_test_logs/log.json
  format: json
  severity: trace
file-cache:
  max-size-mb: $cache_size_mb
  cache-file-for-range-read: $enable_range_read_cache
metadata-cache:
  stat-cache-max-size-mb: 4
  ttl-secs: $cache_ttl
  type-cache-max-size-mb: 32
cache-dir: /tmp/cache-dir" > /tmp/gcsfuse_config.yaml
}

# Read-cache test with cache-file-for-range-read:false.
read_cache_test_setup 50 false 3600
test_cases=(
  "TestCacheFileForRangeReadFalseTest/TestRangeReadsWithCacheMiss"
  "TestCacheFileForRangeReadFalseTest/TestConcurrentReads_ReadIsTreatedNonSequentialAfterFileIsRemovedFromCache"
)
for test_case in "${test_cases[@]}"; do
  gcsfuse --config-file=/tmp/gcsfuse_config.yaml "$TEST_BUCKET_NAME" "$MOUNT_DIR" > /dev/null
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$test_case"
  sudo umount "$MOUNT_DIR"
  cleanup_test_environment
done

# Read-cache test with cache-file-for-range-read:true.
read_cache_test_setup 50 true 3600
test_case="TestCacheFileForRangeReadTrueTest/TestRangeReadsWithCacheHit"
gcsfuse --config-file=/tmp/gcsfuse_config.yaml "$TEST_BUCKET_NAME" "$MOUNT_DIR" > /dev/null
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$test_case"
sudo umount "$MOUNT_DIR"
cleanup_test_environment


# Read-cache test with disabled cache ttl.
read_cache_test_setup 9 false 0
test_case="TestDisabledCacheTTLTest/TestReadAfterObjectUpdateIsCacheMiss"
gcsfuse --config-file=/tmp/gcsfuse_config.yaml "$TEST_BUCKET_NAME" "$MOUNT_DIR" > /dev/null
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$test_case"
sudo umount "$MOUNT_DIR"
cleanup_test_environment

# Read-cache test with local modification.
read_cache_test_setup 9 false 3600
test_case="TestLocalModificationTest/TestReadAfterLocalGCSFuseWriteIsCacheMiss"
gcsfuse --config-file=/tmp/gcsfuse_config.yaml "$TEST_BUCKET_NAME" "$MOUNT_DIR" > /dev/null
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$test_case"
sudo umount "$MOUNT_DIR"
cleanup_test_environment

# Read-cache tests for range reads.
test_cases=(
  "TestRangeReadTest/TestRangeReadsWithinReadChunkSize"
  "TestRangeReadTest/TestRangeReadsBeyondReadChunkSizeWithChunkDownloaded"
)
run_range_read_test_with_read_cache() {
  read_cache_test_setup 500 $1 3600
  gcsfuse --config-file=/tmp/gcsfuse_config.yaml "$TEST_BUCKET_NAME" "$MOUNT_DIR" > /dev/null
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$2"
  sudo umount "$MOUNT_DIR"
}
for test_case in "${test_cases[@]}"; do
    # Run range read test with cache-file-for-range-read: false.
    run_range_read_test_with_read_cache false "$test_case"
    # Run range read test with cache-file-for-range-read: true.
    run_range_read_test_with_read_cache true "$test_case"
done

# Read cache tests on read only mount.
test_cases=(
  "TestReadOnlyTest/TestSecondSequentialReadIsCacheHit"
  "TestReadOnlyTest/TestReadFileSequentiallyLargerThanCacheCapacity"
  "TestReadOnlyTest/TestReadFileRandomlyLargerThanCacheCapacity"
  "TestReadOnlyTest/TestReadMultipleFilesMoreThanCacheLimit"
  "TestReadOnlyTest/TestReadMultipleFilesWithinCacheLimit"
)

run_read_only_test_with_read_cache() {
  read_cache_test_setup 9 $1 3600
  gcsfuse --o=ro --config-file=/tmp/gcsfuse_config.yaml "$TEST_BUCKET_NAME" "$MOUNT_DIR" > /dev/null
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$2"
  sudo umount "$MOUNT_DIR"
}

for test_case in "${test_cases[@]}"; do
  # Run read only test with cache-file-for-range-read: false.
  run_read_only_test_with_read_cache false "$test_case"
  # Run read only test with cache-file-for-range-read: true.
  run_read_only_test_with_read_cache true "$test_case"
done

# Read cache tests with small cache ttl.
read_cache_test_setup 9 false 10
test_cases=(
  "TestSmallCacheTTLTest/TestReadAfterUpdateAndCacheExpiryIsCacheMiss"
  "TestSmallCacheTTLTest/TestReadForLowMetaDataCacheTTLIsCacheHit"
)
for test_case in "${test_cases[@]}"; do
  gcsfuse --config-file=/tmp/gcsfuse_config.yaml "$TEST_BUCKET_NAME" "$MOUNT_DIR" > /dev/null
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/read_cache/... -p 1 --integrationTest -v --mountedDirectory="$MOUNT_DIR" --testbucket="$TEST_BUCKET_NAME" -run "$test_case"
  sudo umount "$MOUNT_DIR"
  cleanup_test_environment
done

echo "list:
  enable-empty-managed-folders: true" > /tmp/gcsfuse_config.yaml

# Package managed_folders
# Empty managed folders listing test.
# Run test with static mounting (flags: --implicit-dirs)
gcsfuse --implicit-dirs --config-file=/tmp/gcsfuse_config.yaml $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/managed_folders/... -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR  --testbucket=$TEST_BUCKET_NAME -run TestEnableEmptyManagedFoldersTrue
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs -o config_file=/tmp/gcsfuse_config.yaml
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/managed_folders/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME -run TestEnableEmptyManagedFoldersTrue
sudo umount $MOUNT_DIR

# For GRPC: running only core integration tests.

# Test packages: operations
# Run test with static mounting. (flags: --client-protocol=grpc --implicit-dirs=true)
gcsfuse --client-protocol=grpc --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --client-protocol=grpc --implicit-dirs=true)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=true,client_protocol=grpc
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Test package: implicit_dir
# Run tests with static mounting.  (flags: --client-protocol=grpc --implicit-dirs=true)
gcsfuse --implicit-dirs=true --client-protocol=grpc $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run test with persistent mounting.  (flags: --client-protocol=grpc --implicit-dirs=true)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=true,client_protocol=grpc
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR
