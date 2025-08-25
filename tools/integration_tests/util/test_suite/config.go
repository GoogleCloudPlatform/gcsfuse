// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test_suite

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// BucketType represents the 'compatible' field.
type BucketType struct {
	Flat  bool `yaml:"flat"`
	Hns   bool `yaml:"hns"`
	Zonal bool `yaml:"zonal"`
}

// TestConfig represents the common configuration for test packages.
type TestConfig struct {
	MountedDirectory string       `yaml:"mounted_directory"`
	TestBucket       string       `yaml:"test_bucket"`
	LogFile          string       `yaml:"log_file,omitempty"`
	RunOnGKE         bool         `yaml:"run_on_gke"`
	Configs          []ConfigItem `yaml:"configs"`
}

// ConfigItem defines the variable parts of each test run.
type ConfigItem struct {
	Flags      []string        `yaml:"flags"`
	Compatible map[string]bool `yaml:"compatible"`
	Tpc        bool            `yaml:"tpc,omitempty"`
}

// Config holds all test configurations parsed from the YAML file.
type Config struct {
	ImplicitDir           []TestConfig `yaml:"implicit_dir"`
	ExplicitDir           []TestConfig `yaml:"explicit_dir"`
	ListLargeDir          []TestConfig `yaml:"list_large_dir"`
	WriteLargeFiles       []TestConfig `yaml:"write_large_files"`
	Operations            []TestConfig `yaml:"operations"`
	ReadLargeFiles        []TestConfig `yaml:"read_large_files"`
	ReadOnly              []TestConfig `yaml:"readonly"`
	ReadCache             []TestConfig `yaml:"read_cache"`
	RenameDirLimit        []TestConfig `yaml:"rename_dir_limit"`
	Gzip                  []TestConfig `yaml:"gzip"`
	LocalFile             []TestConfig `yaml:"local_file"`
	LogRotation           []TestConfig `yaml:"log_rotation"`
	ManagedFolders        []TestConfig `yaml:"managed_folders"`
	ConcurrentOperations  []TestConfig `yaml:"concurrent_operations"`
	Benchmarking          []TestConfig `yaml:"benchmarking"`
	StaleHandles          []TestConfig `yaml:"stale_handles"`
	StreamingWrites       []TestConfig `yaml:"streaming_writes"`
	InactiveStreamTimeout []TestConfig `yaml:"inactive_stream_timeout"`
	CloudProfiler         []TestConfig `yaml:"cloud_profiler"`
	KernelListCache       []TestConfig `yaml:"kernel_list_cache"`
	ReadDirPlus           []TestConfig `yaml:"readdirplus"`
	DentryCache           []TestConfig `yaml:"dentry_cache"`
}

// ReadConfigFile returns a Config struct from the YAML file.
func ReadConfigFile(configFilePath string) Config {
	var cfg Config
	if configFilePath != "" {
		configData, err := os.ReadFile(configFilePath)
		if err != nil {
			log.Fatalf("could not read config file %q: %v", configFilePath, err)
		}
		expandedYaml := os.ExpandEnv(string(configData))
		if err := yaml.Unmarshal([]byte(expandedYaml), &cfg); err != nil {
			log.Fatalf("Failed to parse config YAML: %v", err)
		}
	}
	return cfg
}
