TEST_DIR_PARALLEL=(
  "mounting"
  "explicit_dir"
  "read_cache"
  "local_file"
  "log_rotation"
  "gzip"
  "write_large_files"
  "readonly"
  "implicit_dir"
  "operations"
  "read_large_files"
  "rename_dir_limit"
)
for test_dir_np in "${TEST_DIR_PARALLEL[@]}"
do
   test_path_non_parallel="./tools/integration_tests/$test_dir_np"
   # Executing integration tests
   GODEBUG=asyncpreemptoff=1 go test $test_path_non_parallel -p 1 --integrationTest -v --testbucket=brenna-tpc-bucket -timeout 60m --testInstalledPackage
done
