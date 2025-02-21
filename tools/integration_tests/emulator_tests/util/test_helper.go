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
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/setup"
)

// StartProxyServer starts a proxy server as a background process and handles its lifecycle.
//
// It launches the proxy server with the specified configuration and port, logs its output to a file.
func StartProxyServer(configPath string) {
	port := 8020
	out, _ := exec.Command("sh", "-c", fmt.Sprintf("netstat -ant | grep LISTEN | grep %v", port)).Output()
	log.Printf("netstat output is => %v", string(out))
	time.Sleep(2 * time.Minute) // Sleeping for port resets.
	// Start the proxy in the background
	cmd := exec.Command("go", "run", "../proxy_server/.", "--config-path="+configPath)
	logFileForProxyServer, err := os.Create(path.Join(os.Getenv("KOKORO_ARTIFACTS_DIR"), "proxy-"+setup.GenerateRandomString(5)))
	if err != nil {
		log.Fatal("Error in creating log file for proxy server.")
	}
	log.Printf("Proxy server logs are generated with specific filename %s: ", logFileForProxyServer.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
}

func GetListeningProcessIDOnPort(port int) (int, error) {
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port))
	out, err := cmd.CombinedOutput()

	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok && exiterr.ExitCode() == 1 {
			log.Println("No process found on lsof output.")
			return -1, nil // No process found, port is not listening, no error
		}
		// An actual error occurred (e.g., lsof not found)
		return -1, fmt.Errorf("lsof error: %w, output: %s", err, out)
	}

	lines := strings.Split(string(out), "\n")
	log.Println("lsof output now:")
	for _, line := range lines {
		log.Println(line)
	}
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) > 1 && strings.Contains(line, "LISTEN") {
			pid, err := strconv.Atoi(fields[1])
			if err != nil {
				return -1, fmt.Errorf("failed to parse PID: %w", err)
			}
			return pid, nil // Process is listening, no error
		}
	}
	return -1, nil // if lsof returned something, but no LISTEN line, then return -1.
}

// KillProxyServerProcess kills all processes listening on the specified port.
//
// It uses the `lsof` command to identify the processes and sends SIGINT to each of them.
func KillProxyServerProcess(port int) error {
	log.Printf("Killing Proxy Server Process on port: %v", port)
	for {
		pid, err := GetListeningProcessIDOnPort(port)
		if err != nil {
			return err
		}

		if pid == -1 {
			log.Printf("Port is free to use now.")
			return nil // Port is free
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			// Handle the case where the process is not found
			if strings.Contains(err.Error(), "no such process") {
				continue // Process might have already terminated, continue to the next iteration
			}
			return fmt.Errorf("failed to find process: %w", err)
		}

		if err := process.Signal(syscall.SIGINT); err != nil {
			// Handle the case where signaling fails (e.g., process already terminated)
			if strings.Contains(err.Error(), "no such process") ||
				strings.Contains(err.Error(), "os: process already finished") {
				continue // Process might have already terminated, continue to the next iteration
			}
			return fmt.Errorf("failed to kill process: %w", err)
		}

		time.Sleep(1 * time.Second)
	}
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
		if strings.HasPrefix(flag, "--chunk-transfer-timeout-secs=") {
			valueStr := strings.TrimPrefix(flag, "--chunk-transfer-timeout-secs=")
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
