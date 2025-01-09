# Emulator based tests
## go-proxy server
Proxy server, which intercepts [storage-testbench](https://github.com/googleapis/storage-testbench) server and perform pre-defined
retry test.

### Steps to run the test manually
1. Run storage-testbench server by following [this](https://github.com/googleapis/storage-testbench/tree/main?tab=readme-ov-file#initial-set-up) steps.
2. Create test-bucket on server with below command.
```
cat << EOF > test.json
{"name":"test-bucket"}
EOF

# Execute the curl command to create bucket on storagetestbench server.
curl -X POST --data-binary @test.json \
    -H "Content-Type: application/json" \
    "$STORAGE_EMULATOR_HOST/storage/v1/b?project=test-project"
rm test.json    
```
3. Run tests with the current directory as emulator_tests.
```
go test --integrationTest -v --testbucket=test-bucket -timeout 10m
```

### Automated emulator test script
1. Run ./emulator_tests.sh

### Steps to add new tests in the future:
1. Create <feature>_test file [here](https://github.com/GoogleCloudPlatform/gcsfuse/tree/master/tools/integration_tests/emulator_tests).
2. Write tests according to your scenarios. e.g. [write_stall_test](https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/tools/integration_tests/emulator_tests/write_stall_test.go)
