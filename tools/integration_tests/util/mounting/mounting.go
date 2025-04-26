// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mounting

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v2/tools/integration_tests/util/operations"
)

func extractLogFile(flags []string) (string, error) {
	for _, flag := range flags {
		if strings.HasPrefix(flag, "--log-file=") {
			return strings.TrimPrefix(flag, "--log-file="), nil
		}
	}
	for _, flag := range flags {
		if strings.HasPrefix(flag, "log_file=") {
			return strings.TrimPrefix(flag, "log_file="), nil
		}
	}
	return "", fmt.Errorf("cannot find log file")
}

func MountGcsfuse(binaryFile string, flags []string) error {
	return MountGcsfuseWithEnvOverride(binaryFile, flags, nil)
}

func MountGcsfuseWithEnvOverride(binaryFile string, flags []string, envOverride []string) error {
	mountCmd := exec.Command(
		binaryFile,
		flags...,
	)
	mountCmd.Env = envOverride
	logFile, err := extractLogFile(flags)
	if err != nil {
		return err
	}
	// Adding mount command in LogFile
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Could not open logfile")
	}
	// Closing file at the end.
	defer operations.CloseFile(file)

	_, err = file.WriteString(mountCmd.String() + "\n")
	if err != nil {
		fmt.Println("Could not write cmd to logFile")
	}

	output, err := mountCmd.CombinedOutput()
	if err != nil {
		log.Println(mountCmd.String())
		log.Println("Error: ", string(output))
		return fmt.Errorf("cannot mount gcsfuse: %w", err)
	}
	return nil
}
