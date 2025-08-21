// Copyright 2025 Google LLC
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

package rapid_appends

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// //////////////////////////////////////////////////////////////////////
// Test Configurations
// //////////////////////////////////////////////////////////////////////

// testConfig defines a specific GCSfuse configuration for a test run.
type testConfig struct {
	name                string
	isDualMount         bool
	metadataCacheOnRead bool
	fileCache           bool
	primaryMountFlags   []string
	secondaryMountFlags []string
}

// readTestConfigs defines the matrix of configurations for the ReadsTestSuite.
var readTestConfigs = []*testConfig{
	// Single-Mount Scenarios
	{
		name:                "SingleMount_NoCache",
		isDualMount:         false,
		metadataCacheOnRead: false,
		fileCache:           false,
		primaryMountFlags:   []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	{
		name:                "SingleMount_MetadataCache",
		isDualMount:         false,
		metadataCacheOnRead: true,
		fileCache:           false,
		primaryMountFlags:   []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	{
		name:                "SingleMount_FileCache",
		isDualMount:         false,
		metadataCacheOnRead: false,
		fileCache:           true,
		primaryMountFlags:   []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	{
		name:                "SingleMount_MetadataAndFileCache",
		isDualMount:         false,
		metadataCacheOnRead: true,
		fileCache:           true,
		primaryMountFlags:   []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	// Dual-Mount Scenarios
	{
		name:                "DualMount_NoCache",
		isDualMount:         true,
		metadataCacheOnRead: false,
		fileCache:           false,
		primaryMountFlags:   []string{"--enable-rapid-appends=true"},
		secondaryMountFlags: []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	{
		name:                "DualMount_MetadataCache",
		isDualMount:         true,
		metadataCacheOnRead: true,
		fileCache:           false,
		primaryMountFlags:   []string{"--enable-rapid-appends=true"},
		secondaryMountFlags: []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	{
		name:                "DualMount_FileCache",
		isDualMount:         true,
		metadataCacheOnRead: false,
		fileCache:           true,
		primaryMountFlags:   []string{"--enable-rapid-appends=true"},
		secondaryMountFlags: []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
	{
		name:                "DualMount_MetadataAndFileCache",
		isDualMount:         true,
		metadataCacheOnRead: true,
		fileCache:           true,
		primaryMountFlags:   []string{"--enable-rapid-appends=true"},
		secondaryMountFlags: []string{"--enable-rapid-appends=true", "--write-global-max-blocks=-1"},
	},
}

// appendTestConfigs defines the matrix of configurations for the AppendsTestSuite.
var appendTestConfigs = []*testConfig{
	{
		name:              "SingleMount_BufferedWrite",
		isDualMount:       false,
		primaryMountFlags: []string{"--enable-rapid-appends=true", "--write-block-size-mb=1"},
	},
	{
		name:                "DualMount_BufferedWrite",
		isDualMount:         true,
		primaryMountFlags:   []string{"--enable-rapid-appends=true", "--write-block-size-mb=1"},
		secondaryMountFlags: []string{"--enable-rapid-appends=true", "--write-block-size-mb=1"},
	},
}

// //////////////////////////////////////////////////////////////////////
// Test Runners
// //////////////////////////////////////////////////////////////////////

// TestReadsSuiteRunner executes all read-after-append tests against the readTestConfigs matrix.
func TestReadsSuiteRunner(t *testing.T) {
	for _, cfg := range readTestConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			suite.Run(t, &ReadsTestSuite{BaseSuite{cfg: cfg}})
		})
	}
}

// TestAppendsSuiteRunner executes all general append tests against the appendTestConfigs matrix.
func TestAppendsSuiteRunner(t *testing.T) {
	for _, cfg := range appendTestConfigs {
		t.Run(cfg.name, func(t *testing.T) {
			suite.Run(t, &AppendsTestSuite{BaseSuite{cfg: cfg}})
		})
	}
}
