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
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=true)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run test with static mounting. (flags: --implicit-dirs=false)
gcsfuse --implicit-dirs=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=false)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o implicit_dirs=false
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run tests with --only-dir mounting. (flags: --implicit-dirs=true)
gcsfuse --only-dir testDir --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run tests with persistent mounting with --only-dir flag. (flags: --implicit-dirs=true, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run tests with --only-dir mounting. (flags: --implicit-dirs=false)
gcsfuse --only-dir testDir --implicit-dirs=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run tests with persistent mounting. (flags: --implicit-dirs=false, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,implicit_dirs=false
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
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

# Run tests with --only-dir mounting. (flags: --implicit-dirs=true,--o=ro)
gcsfuse --only-dir testDir --o=ro --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run test with persistent mounting.  (flags: --implicit-dirs=true,--o=ro,--only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o ro,only_dir=testDir,implicit_dirs=true
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run test with --only-dir mounting. (flags: --implicit-dirs=true, --file-mode=544, --dir-mode=544)
gcsfuse --only-dir testDir --file-mode=544 --dir-mode=544 --implicit-dirs=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run test with persistent mounting. (flags: --implicit-dirs=true, --file-mode=544, --dir-mode=544, --only-dir=testDir)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,file_mode=544,dir_mode=544,implicit_dirs=true
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

# Run test with --only-dir mounting. (flags: --rename-dir-limit=3, --implicit-dirs)
gcsfuse --only-dir testDir --rename-dir-limit=3 --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run test with persistent mounting . (flags: --rename-dir-limit=3, --implicit-dirs)
mount.gcsfuse $TEST_BUCKET_NAME $MOUNT_DIR -o only_dir=testDir,rename_dir_limit=3,implicit_dirs
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run test with --only-dir mounting. (flags: --rename-dir-limit=3)
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

# Run tests with --only-dir mounting. (flags: --implicit-dirs)
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

# Run tests with --only-dir mounting. (flags: --implicit-dirs=false)
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

# write_large_files
# Run tests with static mounting. (flags: --implicit-dirs)
gcsfuse --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/write_large_files/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR
