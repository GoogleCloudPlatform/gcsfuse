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
	"path"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

func MountGcsfuse(binaryFile string, flags []string) error {
	mountCmd := exec.Command(
		binaryFile,
		flags...,
	)

	// Adding mount command in LogFile
	err := os.MkdirAll(path.Dir(setup.LogFile()), 0777)
	if err != nil {
		fmt.Println("error creating directory: ", err)
		return err
	}
	file, err := os.OpenFile(setup.LogFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Could not open logfile: ", err.Error())
	}
	// Closing file at the end.
	defer operations.CloseFile(file)

	_, err = file.WriteString(mountCmd.String() + "\n")
	if err != nil {
		fmt.Println("Could not write cmd to logFile: ", err.Error())
	}

	output, err := mountCmd.CombinedOutput()
	if err != nil {
		log.Println(mountCmd.String())
		log.Println("Error: ", string(output))
		return fmt.Errorf("cannot mount gcsfuse: %w\n", err)
	}

	readAheadKB := setup.ReadAheadKB()
	if readAheadKB > 0 {
		mountDir := getMountDir(flags)
		if mountDir == "" {
			mountDir = setup.MntDir()
		}
		if mountDir != "" {
			if err := ConfigureReadAhead(mountDir, readAheadKB); err != nil {
				return fmt.Errorf("failed to configure read-ahead: %w", err)
			}
		} else {
			log.Printf("Warning: read-ahead-kb specified but mount directory could not be resolved: %v", flags)
		}
	}

	return nil
}

func getMountDir(flags []string) string {
	for i := len(flags) - 1; i >= 0; i-- {
		arg := flags[i]
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if fi, err := os.Stat(arg); err == nil && fi.IsDir() {
			return arg
		}
	}
	return ""
}
