//Copyright 2023 Google Inc. All Rights Reserved.
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package multiple_mounting

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/operations"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func getLogFilePaths(numOfMounts int) (logFilePaths []string) {
	logFilePaths = make([]string, numOfMounts)
	for i := 0; i < numOfMounts; i++ {
		logFilePaths[i] = path.Join(setup.TestDir(), "gcsfuse"+strconv.Itoa(i)+".log")
	}
	return
}

func mountMultipleGCSfuseMounts(flags []string, mountPaths []string) (err error) {
	commonFlags := []string{"--debug_gcs",
		"--debug_fs",
		"--debug_fuse",
		"--log-format=text"}

	for i := 0; i < len(commonFlags); i++ {
		flags = append(flags, commonFlags[i])
	}

	// Unmount the mounts in case of failure mounting any one of the mounts.
	shouldUnMount := false
	unMount := func(mountPath string) {
		if shouldUnMount {
			err := mounting.UnMountGcsfuse(mountPath)
			if err != nil {
				log.Printf("failure in unmounting: %s\n", err.Error())
			}
			operations.RemoveDir(mountPath)
			return
		}
	}

	logFilePaths := getLogFilePaths(len(mountPaths))
	for i := 0; i < len(mountPaths); i++ {
		// Add mount path to flags
		mountFlags := make([]string, len(flags)+3)
		copy(mountFlags, flags)
		// set separate log file, bucket name and mount path
		mountFlags[len(flags)] = "--log-file=" + logFilePaths[i]
		mountFlags[len(flags)+1] = setup.TestBucket()
		mountFlags[len(flags)+2] = mountPaths[i]
		err := os.Mkdir(mountPaths[i], 0755)
		if err != nil {
			shouldUnMount = true
			return err
		}
		err = mounting.MountGcsfuse(setup.BinFile(), mountFlags, setup.LogFile())
		if err != nil {
			shouldUnMount = true
			return err
		}
		defer unMount(mountPaths[i])
	}

	return err
}

func executeTestsForMultipleMounting(flags [][]string, mountPaths []string, m *testing.M) (successCode int) {
	var err error

	for i := 0; i < len(flags); i++ {
		setup.CleanMntDir()
		if err = mountMultipleGCSfuseMounts(flags[i], mountPaths); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountMultipleGCSfuseMounts: %v\n", err))
		}

		successCode = setup.ExecuteTest(m)

		var unmountingFailed bool
		for i := 0; i < len(mountPaths); i++ {
			err := mounting.UnMountGcsfuse(mountPaths[i])
			if err != nil {
				unmountingFailed = true
				log.Printf("Error while unMounting %s: %s", mountPaths[i], err.Error())
			}
		}

		if successCode != 0 {
			// Print flag on which test failed.
			f := strings.Join(flags[i], " ")
			log.Print("Test Fails on " + f)
		}

		if unmountingFailed {
			setup.LogAndExit("Error in unmounting one or more bucket")
		}
	}
	return
}

func RunTests(flags [][]string, mountPaths []string, m *testing.M) (successCode int) {
	successCode = executeTestsForMultipleMounting(flags, mountPaths, m)

	log.Printf("GCSFuse log files(gcsfuse*.log) under: %s\n", setup.TestDir())

	return successCode
}
