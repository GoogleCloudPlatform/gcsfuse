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

	"github.com/googlecloudplatform/gcsfuse/v3/tools/integration_tests/util/setup"
)

// FilterLogFileFlags removes any existing --log-file or log_file flags from the flags slice.
func FilterLogFileFlags(flags []string) []string {
	var filtered []string
	for _, flag := range flags {
		if strings.HasPrefix(flag, "--log-file=") || strings.HasPrefix(flag, "log_file=") {
			continue
		}
		filtered = append(filtered, flag)
	}
	return filtered
}

func MountGcsfuse(binaryFile string, flags []string) error {
	return MountGcsfuseWithEnv(binaryFile, flags, nil, setup.LogFile())
}

func MountGcsfuseWithEnv(binaryFile string, flags []string, env map[string]string, logFile string) error {
	if binaryFile == "" {
		binaryFile = setup.BinFile()
	}
	if logFile == "" {
		return fmt.Errorf("logFile is required")
	}

	mountCmd := exec.Command(
		binaryFile,
		flags...,
	)
	if len(env) > 0 {
		mountCmd.Env = os.Environ()
		for k, v := range env {
			mountCmd.Env = append(mountCmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	err := os.MkdirAll(path.Dir(logFile), 0777)
	if err != nil {
		fmt.Println("error creating directory: ", err)
		return err
	}
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Could not open logfile: ", err.Error())
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("error closing logfile %s: %v", logFile, err)
		}
	}()

	_, err = file.WriteString(mountCmd.String() + "\n")
	if err != nil {
		fmt.Println("Could not write cmd to logFile: ", err.Error())
		return err
	}

	output, err := mountCmd.CombinedOutput()
	if err != nil {
		log.Println(mountCmd.String())
		log.Println("Error: ", string(output))
		return fmt.Errorf("cannot mount gcsfuse: %w\n", err)
	}
	return nil
}
