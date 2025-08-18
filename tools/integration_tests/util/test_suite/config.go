// Copyright 2025 Google LLC
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

package test_suite

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
}
