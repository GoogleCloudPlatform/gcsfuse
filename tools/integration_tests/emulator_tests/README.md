# go-proxy-server
Proxy server, which intercepts storage-test-bench server and perform pre-defined
retry test.

### Steps to run the test
1. Run storage-testbench by following [this](https://github.com/googleapis/storage-testbench/tree/main?tab=readme-ov-file#initial-set-up) steps.
2. Run the proxy server: `go run .` This will start the proxy server at `localhost:8080`.
3. Run the test: `STORAGE_EMULATOR_HOST=http://localhost:8080 go test . -v -count=1`

### Automated emulator test
1. Run ./emulator_test.sh
