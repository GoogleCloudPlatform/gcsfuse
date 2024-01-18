// Copyright 2023 Google Inc. All Rights Reserved.
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

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func MountGcsfuse(binaryFile string, flags []string) error {
	mountCmd := exec.Command(
		binaryFile,
		flags...,
	)

	// Adding mount command in LogFile
	file, err := os.OpenFile(setup.LogFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
		return fmt.Errorf("cannot mount gcsfuse: %w\n", err)
	}
	return nil
}
