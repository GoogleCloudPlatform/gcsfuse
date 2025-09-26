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
	"path"

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
	GKEMountedDirectory     string `yaml:"mounted_directory"`
	GCSFuseMountedDirectory string
	TestBucket              string       `yaml:"test_bucket"`
	LogFile                 string       `yaml:"log_file,omitempty"`
	Configs                 []ConfigItem `yaml:"configs"`
	OnlyDir                 string       `yaml:"only_dir,omitempty"`
}

// ConfigItem defines the variable parts of each test run.
type ConfigItem struct {
	Flags      []string        `yaml:"flags"`
	Compatible map[string]bool `yaml:"compatible"`
	Run        string          `yaml:"run,omitempty"`
	RunOnGKE   bool            `yaml:"run_on_gke"`
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
	RequesterPaysBucket   []TestConfig `yaml:"requester_pays_bucket"`
	ReadGCSAlgo           []TestConfig `yaml:"read_gcs_algo"`
	Interrupt             []TestConfig `yaml:"interrupt"`
	UnfinalizedObject     []TestConfig `yaml:"unfinalized_object"`
	RapidAppends          []TestConfig `yaml:"rapid_appends"`
	MountTimeout          []TestConfig `yaml:"mount_timeout"`
	Monitoring            []TestConfig `yaml:"monitoring"`
	FlagOptimizations     []TestConfig `yaml:"flag_optimizations"`
}

func processTestConfigs(configs []TestConfig) {
	for i := range configs {
		if configs[i].OnlyDir != "" && configs[i].GKEMountedDirectory != "" {
			// Add onlyDir infront of bucket_name incase of mounted dir
			configs[i].TestBucket = path.Join(configs[i].TestBucket, configs[i].OnlyDir)
		}
	}
}

func (c *Config) postProcessConfig() {
	processTestConfigs(c.ImplicitDir)
	processTestConfigs(c.ExplicitDir)
	processTestConfigs(c.ListLargeDir)
	processTestConfigs(c.WriteLargeFiles)
	processTestConfigs(c.Operations)
	processTestConfigs(c.ReadLargeFiles)
	processTestConfigs(c.ReadOnly)
	processTestConfigs(c.ReadCache)
	processTestConfigs(c.RenameDirLimit)
	processTestConfigs(c.Gzip)
	processTestConfigs(c.LocalFile)
	processTestConfigs(c.LogRotation)
	processTestConfigs(c.ManagedFolders)
	processTestConfigs(c.ConcurrentOperations)
	processTestConfigs(c.Benchmarking)
	processTestConfigs(c.StaleHandles)
	processTestConfigs(c.StreamingWrites)
	processTestConfigs(c.InactiveStreamTimeout)
	processTestConfigs(c.CloudProfiler)
	processTestConfigs(c.KernelListCache)
	processTestConfigs(c.ReadDirPlus)
	processTestConfigs(c.DentryCache)
	processTestConfigs(c.ReadGCSAlgo)
	processTestConfigs(c.Interrupt)
	processTestConfigs(c.UnfinalizedObject)
	processTestConfigs(c.RapidAppends)
	processTestConfigs(c.MountTimeout)
	processTestConfigs(c.Monitoring)
	processTestConfigs(c.FlagOptimizations)
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

	cfg.postProcessConfig()
	return cfg
}
