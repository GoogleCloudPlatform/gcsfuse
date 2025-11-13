// Copyright 2023 Google LLC
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

// Provide helper functions.
package operations

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"time"
)

// GenerateRandomData generates random data that can be used to write to a file.
func GenerateRandomData(sizeInBytes int64) ([]byte, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	data := make([]byte, sizeInBytes)
	_, err := r.Read(data)
	if err != nil {
		return nil, fmt.Errorf("r.Read(): %v", err)
	}
	return data, nil
}

func setEnvVariables(cmd *exec.Cmd) {
	cmd.Env = os.Environ() // Start with existing environment
	cmd.Env = append(cmd.Env, "CLOUDSDK_PYTHON=$HOME/.local/python-3.11.9/bin/python3.11")
	cmd.Env = append(cmd.Env, "PATH=$HOME/.local/python-3.11.9/bin:$PATH")
	cmd.Env = append(cmd.Env, "PATH=/usr/local/google-cloud-sdk/bin:"+os.Getenv("PATH")) // Ensure latest gcloud bin is first
}

// Executes any given tool (e.g. gsutil/gcloud) with given args.
func executeToolCommandf(tool string, format string, args ...any) ([]byte, error) {
	cmdArgs := tool + " " + fmt.Sprintf(format, args...)
	cmd := exec.Command("/bin/bash", "-c", cmdArgs)

	return runCommand(cmd)
}

// Executes any given tool (e.g. gsutil/gcloud).
func executeToolCommand(tool string, command string) ([]byte, error) {
	cmdArgs := tool + " " + command
	cmd := exec.Command("/bin/bash", "-c", cmdArgs)

	return runCommand(cmd)
}

// Executes any given tool (e.g. gsutil/gcloud) with given args in specified directory.
func ExecuteToolCommandfInDirectory(dirPath, tool, format string, args ...any) ([]byte, error) {
	cmdArgs := tool + " " + fmt.Sprintf(format, args...)
	cmd := exec.Command("/bin/bash", "-c", cmdArgs)
	cmd.Dir = dirPath

	return runCommand(cmd)
}

func runCommand(cmd *exec.Cmd) ([]byte, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	setEnvVariables(cmd)

	err := cmd.Run()
	if err != nil {
		return stdout.Bytes(), fmt.Errorf("failed command '%s': %v, %s", cmd.String(), err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// ExecuteGcloudCommandf executes any given gcloud command with given args.
func ExecuteGcloudCommandf(format string, args ...any) ([]byte, error) {
	return executeToolCommandf("gcloud", format, args...)
}

// ExecuteGcloudCommand executes any given gcloud command.
func ExecuteGcloudCommand(command string) ([]byte, error) {
	return executeToolCommand("gcloud", command)
}

// WaitForSizeUpdate waits for a specified time duration to ensure that stat()
// call returns correct size for unfinalized object.
func WaitForSizeUpdate(isZonal bool, duration time.Duration) {
	if isZonal {
		time.Sleep(duration)
	}
}
