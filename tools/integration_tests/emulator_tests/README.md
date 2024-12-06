# go-proxy-server
Proxy server, which intercepts storage-test-bench server and perform pre-defined
retry test.

### Steps to run the test
1. Run storage-testbench by following [this](https://github.com/googleapis/storage-testbench/tree/main?tab=readme-ov-file#initial-set-up) steps.
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
2. Run the proxy server: `go run . --config-path=<file>` This will start the proxy server at `localhost:8020`.
3. Run the test: `STORAGE_EMULATOR_HOST="http://localhost:8020" go test --integrationTest -v --testbucket=test-bucket -timeout 10m -run $test_name`

### Automated emulator test
1. Run ./emulator_tests.sh

### Steps to add new tests in the future:
1. Create a new directory for your test.
2. Add a YAML file to the [configs](https://github.com/GoogleCloudPlatform/gcsfuse/tree/master/tools/integration_tests/emulator_tests/proxy_server/configs) directory to create a forced retry scenario.
3. Add the YAML file and package name pair to the emulator_tests.sh file.
