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

// Provide tests for explicit directory only.
package explicit_dir_test

import (
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/internal/config"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup"
	"github.com/googlecloudplatform/gcsfuse/tools/integration_tests/util/setup/implicit_and_explicit_dir_setup"
)

func createMountConfigsAndEquivalentFlags() (flags [][]string) {
	flags = [][]string{{}}

	// base case with only --implicit-dirs=false
	flags = append(flags, []string{"--implicit-dirs=false"})

	// advanced case with --implicit-dirs=false and metadata-cache configuration
	mountConfig := config.MountConfig{
		MetadataCacheConfig: config.MetadataCacheConfig{
			TtlInSeconds:                   21600,
			TypeCacheMaxSizeMbPerDirectory: 32,
		},
		LogConfig: config.LogConfig{
			Severity:        config.TRACE,
			LogRotateConfig: config.DefaultLogRotateConfig(),
		},
	}
	filePath := setup.YAMLConfigFile(mountConfig, "config.yaml")
	flags = append(flags, []string{"--implicit-dirs=false", "--config-file=" + filePath})

	return flags
}

func TestMain(m *testing.M) {
	setup.ParseSetUpFlags()

	flags := createMountConfigsAndEquivalentFlags()

	successCode := implicit_and_explicit_dir_setup.RunTestsForImplicitDirAndExplicitDir(flags, m)

	os.Exit(successCode)
}
