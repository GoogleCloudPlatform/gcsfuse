// Copyright 2015 Google Inc. All Rights Reserved.
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

	"github.com/googlecloudplatform/gcsfuse/v2/cfg"
	"github.com/googlecloudplatform/gcsfuse/v2/cmd"
	"github.com/googlecloudplatform/gcsfuse/v2/internal/logger"
)

func logPanic() {
	// Detect if panic happens in main go routine.
	a := recover()
	if a != nil {
		logger.Fatal("Panic: %v", a)
	}
}

// convertToPosixArgs converts a slice of commandline args and transforms them
// into POSIX compliant args. All it does is that it converts flags specified
// using a single-hyphen to double-hyphens. We are excluding "-v" because it's
// reserved for showing version in Cobra.
func convertToPosixArgs(args []string) []string {
	pArgs := make([]string, 0, len(args))
	for _, a := range args {
		if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && a != "-v" {
			pArgs = append(pArgs, "-"+a)
		} else {
			pArgs = append(pArgs, a)
		}
	}
	return pArgs
}

// Don't remove the comment below. It's used to generate config.go file
// which is used for flag and config file parsing.
// Refer https://go.dev/blog/generate for details.
//
//go:generate go run -C tools/config-gen . --paramsFile=params.yaml --outFile=../../cfg/config.go --templateFile=config.tpl
func main() {
	// Common configuration for all commands
	defer logPanic()
	// Make logging output better.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	if strings.ToLower(os.Getenv("ENABLE_GCSFUSE_VIPER_CONFIG")) == "true" {
		// TODO: implement the mount logic instead of simply returning nil.
		rootCmd, err := cmd.NewRootCmd(func(config cfg.Config) error { return nil })
		if err != nil {
			log.Fatalf("Error occurred while creating the root command: %v", err)
		}
		rootCmd.SetArgs(convertToPosixArgs(os.Args))
		if err := rootCmd.Execute(); err != nil {
			log.Fatalf("Error occurred during command execution: %v", err)
		}
		return
	}
	cmd.ExecuteLegacyMain()
}
