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

package shared

// ProfileOptimization holds the rules for a single performance profile.
type ProfileOptimization struct {
	Name  string `yaml:"name"`
	Value any    `yaml:"value"`
}

// MachineBasedOptimization defines a machine-group-based optimization.
type MachineBasedOptimization struct {
	Group string `yaml:"group"`
	Value any    `yaml:"value"`
}

// BucketBasedOptimization defines a bucket-type-based optimization.
type BucketBasedOptimization struct {
	BucketType string `yaml:"bucket-type"`
	Value      any    `yaml:"value"`
}

// OptimizationRules holds all defined optimizations for a single flag.
type OptimizationRules struct {
	MachineBasedOptimization []MachineBasedOptimization `yaml:"machine-based-optimization"`
	BucketBasedOptimization  []BucketBasedOptimization  `yaml:"bucket-based-optimization"`
	Profiles                 []ProfileOptimization      `yaml:"profiles"`
}
