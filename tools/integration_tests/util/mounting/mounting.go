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
	"strconv"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v3/internal/kernelparams"
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

	// Apply read-ahead settings if configured
	if setup.ReadAheadKb() != "" && setup.ReadAheadKb() != "default" {
		kb, err := strconv.Atoi(setup.ReadAheadKb())
		if err != nil {
			log.Fatalf("Invalid value for -read-ahead-kb: %q. Must be an integer or 'default'", setup.ReadAheadKb())
		}
		if kb >= 0 {
			mountPoint := getMountPointFromFlags(binaryFile, flags)
			if mountPoint != "" {
				manager := kernelparams.NewKernelParamsManager()
				manager.SetReadAheadKb(kb)
				manager.ApplyNonGKE(mountPoint)
				log.Printf("Applied read_ahead_kb setting of %d KiB to mount point: %s", kb, mountPoint)
			}
		}
	}

	return nil
}

func getMountPointFromFlags(binaryFile string, flags []string) string {
	if len(flags) < 2 {
		return ""
	}
	// Check if this is the persistent mount helper (sbin/mount.gcsfuse)
	if strings.Contains(binaryFile, "mount.gcsfuse") {
		return flags[1]
	}
	// Otherwise, it's standard gcsfuse, so the mount point is the last argument
	return flags[len(flags)-1]
}
