// Copyright 2023 Google Inc. All Rights Reserved.
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

// Executes any given tool (e.g. gsutil/gcloud) with given args.
func executeToolCommandf(tool string, format string, args ...any) ([]byte, error) {
	cmdArgs := tool + " " + fmt.Sprintf(format, args...)
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

	err := cmd.Run()
	if err != nil {
		return stdout.Bytes(), fmt.Errorf("failed command '%s': %v, %s", cmd.String(), err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// Executes any given gcloud command with given args.
func ExecuteGcloudCommandf(format string, args ...any) ([]byte, error) {
	return executeToolCommandf("gcloud", format, args...)
}
