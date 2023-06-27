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

# Run integration tests for operations directory with static mounting
gcsfuse --enable-storage-client-library=true --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

gcsfuse --enable-storage-client-library=false  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

gcsfuse  --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

gcsfuse --implicit-dirs=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run integration tests for operations with --only-dir mounting.
gcsfuse --only-dir testDir --enable-storage-client-library=true --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

gcsfuse --only-dir testDir --enable-storage-client-library=false  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

gcsfuse  --only-dir testDir --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

gcsfuse --only-dir testDir --implicit-dirs=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run integration tests for readonly directory with static mounting
gcsfuse --o=ro --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

gcsfuse --file-mode=544 --dir-mode=544 --implicit-dirs=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run integration tests for readonly with --only-dir mounting.
gcsfuse --only-dir testDir --o=ro --implicit-dirs=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

gcsfuse --only-dir testDir --file-mode=544 --dir-mode=544 --implicit-dirs=true  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/readonly/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run integration tests for rename_dir_limit directory with static mounting
gcsfuse --rename-dir-limit=3 --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

gcsfuse --rename-dir-limit=3  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run integration tests for rename_dir_limit with --only-dir mounting.
gcsfuse --only-dir testDir --rename-dir-limit=3 --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

gcsfuse --only-dir testDir --rename-dir-limit=3  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/rename_dir_limit/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR
sudo umount $MOUNT_DIR

# Run integration tests for implicit_dir directory with static mounting
gcsfuse --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

gcsfuse --enable-storage-client-library=false --implicit-dirs  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run integration tests for implicit_dir with --only-dir mounting.
gcsfuse --only-dir testDir --implicit-dirs $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

gcsfuse --only-dir testDir --enable-storage-client-library=false --implicit-dirs  $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/implicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

# Run integration tests for explicit_dir directory with static mounting
gcsfuse --enable-storage-client-library=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/explicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

gcsfuse --enable-storage-client-library=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/explicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME
sudo umount $MOUNT_DIR

# Run integration tests for explicit_dir with --only-dir mounting.
gcsfuse --only-dir testDir --enable-storage-client-library=true $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/explicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR

gcsfuse --only-dir testDir --enable-storage-client-library=false $TEST_BUCKET_NAME $MOUNT_DIR
GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/explicit_dir/...  -p 1 --integrationTest -v --mountedDirectory=$MOUNT_DIR --testbucket=$TEST_BUCKET_NAME/testDir
sudo umount $MOUNT_DIR
