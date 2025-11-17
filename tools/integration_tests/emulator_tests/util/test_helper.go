// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package emulator_tests

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/vipnydav/gcsfuse/v3/tools/integration_tests/util/operations"
)

const PortAndProxyProcessIdInfoRegex = `Listening Proxy Server On Port \[(\d+)\] with Process ID \[(\d+)\]`

// StartProxyServer starts the proxy server in the background and returns the port and process ID it is listening on.
//
// It takes the config path and log file path as input. The proxy server is starts in background using
// the go run command. The output of the proxy server is redirected to the log file. It also sets the
// STORAGE_EMULATOR_HOST to `localhost:port` so that any new storage client will use this endpoint by
// default. If any error occurs it fails.
//
// Parameters:
//   - configPath: Path for config for Proxy Server.
//   - logFilePath: Path for log file for Proxy Server Logs.
//
// Returns:
//   - int: Port number that the proxy server is listening on.
//   - int: Proxy Server Process ID.
//   - error: An error if any error occurs in setting up Proxy Server.
func StartProxyServer(configPath, logFilePath string) (int, int, error) {
	cmd := exec.Command("go", "run", "../../../proxy_server/.", "--config-path="+configPath, "--log-file="+logFilePath)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err := cmd.Start()
	if err != nil {
		return -1, -1, fmt.Errorf("error executing proxy server cmd: %v", err)
	}
	port, proxyProcessId, err := getPortAndProcessInfoFromLogFile(logFilePath)
	if err != nil {
		return -1, -1, fmt.Errorf("error getting port information from log file: %v", err)
	}
	// Set STORAGE_EMULATOR_HOST to current proxy server for the test. This ensures
	// any storage client created during this test will call this proxy endpoint.
	// More details: https://cloud.google.com/go/docs/reference/cloud.google.com/go/storage/latest#hdr-Creating_a_Client
	err = os.Setenv("STORAGE_EMULATOR_HOST", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return -1, -1, fmt.Errorf("error setting STORAGE_EMULATOR_HOST environment variable: %v", err)
	}
	return port, proxyProcessId, nil
}

// KillProxyServerProcess kills given Proxy Server Process ID.
// It also unsets the STORAGE_EMULATOR_HOST environment variable for proxy server.
//
// Parameters:
//   - proxyProcessId: PID of Proxy Server.
//
// Returns:
//   - error: if any error occurs in unsetting env or killing proxy Server.
func KillProxyServerProcess(proxyProcessId int) error {
	// Unset STORAGE_EMULATOR_HOST environment set in StartProxyServer.
	err := os.Unsetenv("STORAGE_EMULATOR_HOST")
	if err != nil {
		return fmt.Errorf("error unsetting STORAGE_EMULATOR_HOST environment variable: %v", err)
	}
	// Find Proxy Server Process.
	process, err := os.FindProcess(proxyProcessId)
	if err != nil {
		return fmt.Errorf("error finding the proxy server process with PID %d: %v", proxyProcessId, err)
	}
	// Send SIGINT to the Process.
	err = process.Signal(syscall.SIGINT)
	if err != nil {
		return fmt.Errorf("error sending SIGINT to the proxy server process with PID %d: %v", proxyProcessId, err)
	}
	return nil
}

// WriteFileAndSync creates a file at the given path, writes random data to it,
// and then syncs the file to GCS. It returns the time taken for the sync operation
// and any error encountered.
//
// This function is useful for testing scenarios where file write and sync operations
// might be subject to delays or timeouts.
//
// Parameters:
//   - filePath: The path where the file should be created.
//   - fileSize: The size of the random data to be written to the file.
//
// Returns:
//   - time.Duration: The elapsed time for the file.Sync() operation.
//   - error: Any error encountered during file creation, writing, or syncing.
func WriteFileAndSync(filePath string, fileSize int) (time.Duration, error) {
	// Create a file for writing
	file, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Generate random data
	data := make([]byte, fileSize)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		return 0, err
	}

	// Write the data to the file
	if _, err := file.Write(data); err != nil {
		return 0, err
	}

	startTime := time.Now()
	err = file.Sync()
	endTime := time.Now()

	if err != nil {
		return 0, err
	}

	return endTime.Sub(startTime), nil
}

// ReadFirstByte reads the first byte of a file and returns the time taken.
func ReadFirstByte(t *testing.T, filePath string) (time.Duration, error) {
	t.Helper()

	file, err := operations.OpenFileAsReadonly(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	buffer := make([]byte, 1)

	startTime := time.Now()
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return 0, err
	}

	endTime := time.Now()

	return endTime.Sub(startTime), nil
}

func GetChunkTransferTimeoutFromFlags(flags []string) (int, error) {
	timeout := 10 // Default value
	for _, flag := range flags {
		if after, ok := strings.CutPrefix(flag, "--chunk-transfer-timeout-secs="); ok {
			valueStr := after
			var err error
			timeout, err = strconv.Atoi(valueStr)
			if err != nil {
				return 0, err
			}
			break
		}
	}
	return timeout, nil
}

// GetPortFromLogFile polls a log file for a specific log message containing the port number.
// It returns the port number if found, or an error if the port is not found within the specified duration.
//
// Parameters:
//   - logFilePath: The path to the log file to monitor.
//
// Returns:
//   - Port, proxy process ID , or an error if any error occurs.
func getPortAndProcessInfoFromLogFile(logFilePath string) (int, int, error) {
	logPollingDuration := 3 * time.Minute  // Duration to poll logs.
	logPollingFrequency := 3 * time.Second // Frequency to poll logs.

	// Regular expression to extract the port number and Process ID from the log file.
	re := regexp.MustCompile(PortAndProxyProcessIdInfoRegex)

	// Calculate the timeout time.
	timeout := time.After(logPollingDuration)

	// Create a ticker to poll the log file at the specified frequency.
	ticker := time.NewTicker(logPollingFrequency)
	defer ticker.Stop()

	// Poll the log file until the timeout or the port is found.
	for {
		select {
		case <-timeout:
			// Timeout occurred, return an error.
			return -1, -1, errors.New("timeout: port number not found in log file")
		case <-ticker.C:
			// Read the log file.
			content, err := operations.ReadFile(logFilePath)
			if err != nil {
				// Log file could not be opened, return an error.
				return -1, -1, fmt.Errorf("failed to read the log file: %w", err)
			}

			// Attempt to extract the port number from the log line.
			match := re.FindStringSubmatch(string(content))
			if len(match) == 3 {
				// Port number and PID found, parse it and return.
				portStr, proxyProcessIdStr := match[1], match[2]

				port, err := strconv.Atoi(portStr)
				if err != nil {
					// Port number could not be parsed, return an error.
					return -1, -1, fmt.Errorf("failed to parse port number: %w", err)
				}
				proxyProcessId, err := strconv.Atoi(proxyProcessIdStr)
				if err != nil {
					// Process ID could not be parsed, return an error.
					return -1, -1, fmt.Errorf("failed to parse process ID: %w", err)
				}
				return port, proxyProcessId, nil
			}
		}
	}
}
