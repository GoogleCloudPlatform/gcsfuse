// Copyright 2015 Google LLC
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

// A fuse file system for Google Cloud Storage buckets.
//
// Usage:
//
//	gcsfuse [flags] bucket mount_point
package main

import (
	"log"
	"os"
	"strings"

	"github.com/googlecloudplatform/gcsfuse/v2/cmd"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/perf"
)

func logPanic() {
	// Detect if panic happens in main go routine.
	a := recover()
	if a != nil {
		logger.Fatal("Panic: %v", a)
	}
}

// Don't remove the comment below. It's used to generate config.go file
// which is used for flag and config file parsing.
// Refer https://go.dev/blog/generate for details.
//
//go:generate go run -C tools/config-gen . --paramsFile=../../cfg/params.yaml --outDir=../../cfg --templateDir=templates
func main() {
	// Common configuration for all commands
	defer logPanic()
	// Make logging output better.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	// Set up profiling handlers.
	go perf.HandleCPUProfileSignals()
	go perf.HandleMemoryProfileSignals()

	// TODO: Clean this up after we gain enough confidence on CLI-Config Parity changes.
	disableViperConfigFlag := "disable-viper-config"
	var newOsArgs []string
	for _, arg := range os.Args {
		if arg == "-"+disableViperConfigFlag || arg == "--"+disableViperConfigFlag || arg == "-"+disableViperConfigFlag+"=true" || arg == "--"+disableViperConfigFlag+"=true" {
			err := os.Setenv(cmd.EnableViperConfigEnvVariable, "false")
			if err != nil {
				logger.Infof("error while setting "+cmd.EnableViperConfigEnvVariable+" environment variable: %v", err)
			}
		}
		if !strings.Contains(arg, disableViperConfigFlag) {
			newOsArgs = append(newOsArgs, arg)
		}
	}
	os.Args = newOsArgs

	if strings.ToLower(os.Getenv(cmd.EnableViperConfigEnvVariable)) == "false" {
		cmd.ExecuteLegacyMain()
		return
	}

	cmd.ExecuteNewMain()
}
