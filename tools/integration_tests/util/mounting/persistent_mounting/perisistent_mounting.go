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

package persistent_mounting

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/mounting"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
)

func makePersistentMountingArgs(flags []string) (args []string, err error) {
	// Deal with options.
	for i := range flags {
		switch flags[i] {
		// Don't pass through options that are relevant to mount(8) but not to
		// gcsfuse, and that fusermount chokes on with "Invalid argument" on Linux.
		case "user", "nouser", "auto", "noauto", "_netdev", "no_netdev":

		// Special case: support mount-like formatting for gcsfuse bool flags.
		case "--implicit-dirs",
			"--foreground",
			"--experimental-local-file-cache",
			"--enable-storage-client-library",
			"--reuse-token-from-url",
			"--enable-nonexistent-type-cache":

			s := strings.Replace(flags[i], "-", "_", -1)
			s = strings.Replace(s, "-", "", -1)
			args = append(args, s)

		// Special case: support mount-like formatting for gcsfuse string flags.
		case "--dir-mode",
			"--file-mode",
			"--uid",
			"--gid",
			"--app-name",
			"--only-dir",
			"--billing-project",
			"--client-protocol",
			"--key-file",
			"--token-url",
			"--limit-bytes-per-sec",
			"--limit-ops-per-sec",
			"--rename-dir-limit",
			"--max-retry-sleep",
			"--max-retry-duration",
			"--retry-multiplier",
			"--stat-cache-capacity",
			"--stat-cache-ttl",
			"--type-cache-ttl",
			"--http-client-timeout",
			"--sequential-read-size-mb",
			"--temp-dir",
			"--max-conns-per-host",
			"--max-idle-conns-per-host",
			"--stackdriver-export-interval",
			"--experimental-opentelemetry-collector-address",
			"--log-format",
			"--log-file",
			"--endpoint":
			s := strings.Replace(flags[i], "-", "_", -1)
			s = strings.Replace(s, "-", "", -1)
			args = append(args, s)

		// Pass through everything else.
		default:
			args = append(args, flags[i])
		}
	}
	return
}

func mountGcsfuseWithStaticMounting(flags []string) (err error) {
	defaultArg := []string{setup.TestBucket(),
		setup.MntDir(),
		"-o",
		"debug_gcs",
		"-o",
		"debug_fs",
		"-o",
		"debug_fuse",
		"-o",
		"log_file=" + setup.LogFile(),
		"log_format=text",
	}

	persistentMountingArgs, err := makePersistentMountingArgs(flags)
	if err != nil {
		setup.LogAndExit("Error in converting flags for persistent mounting.")
	}
	for i := 0; i < len(defaultArg); i++ {
		persistentMountingArgs = append(persistentMountingArgs, defaultArg[i])
	}

	setup.SetBinFile("mount -t " + setup.BinFile())

	err = mounting.MountGcsfuse(persistentMountingArgs)

	return err
}

func executeTestsForStatingMounting(flags [][]string, m *testing.M) (successCode int) {
	var err error

	for i := 0; i < len(flags); i++ {
		if err = mountGcsfuseWithStaticMounting(flags[i]); err != nil {
			setup.LogAndExit(fmt.Sprintf("mountGcsfuse: %v\n", err))
		}
		setup.ExecuteTestForFlagsSet(flags[i], m)
	}
	return
}

func RunTests(flags [][]string, m *testing.M) (successCode int) {
	successCode = executeTestsForStatingMounting(flags, m)

	log.Printf("Test log: %s\n", setup.LogFile())

	setup.SetBinFile(setup.BinFile())
	
	return successCode
}
