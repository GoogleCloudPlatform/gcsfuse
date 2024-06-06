for i in {1..100}
do
  echo "Running iteration $i"
  GODEBUG=asyncpreemptoff=1 go test ./tools/integration_tests/operations/...  -p 1 --integrationTest -v  --testbucket=tulsishah_test -run TestParallelLookUpAndDeleteSameFile
done
