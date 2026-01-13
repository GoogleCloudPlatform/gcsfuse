// Copyright 2026 Google LLC
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

package kerneltuner

import (
	"time"

	"github.com/google/uuid"
)

const (
	// CurrentContractVersion helps detect if the producer and consumer are out of sync.
	//
	// BREAKING CHANGES (Increment this version):
	// 1. Renaming any JSON tag (e.g., changing `json:"request_id"` to `json:"id"`).
	// 2. Removing an existing field from a struct.
	// 3. Changing the data type of a field (e.g., string to int).
	// 4. Changing the string value of an existing ParamName constant.
	//
	// NON-BREAKING CHANGES:
	// 1. Adding a new field with a new JSON tag.
	// 2. Adding a new ParamName constant.
	// Follow this guide to make any changes to this contract: TODO(mohit)
	CurrentContractVersion = "1.0.0"
	// TimeFormat is the rigid layout for parsing.
	TimeFormat = time.RFC3339Nano
)

// ParamName acts as an Enum for the parameter keys to ensure contract safety from typo errors.
type ParamName string

const (
	MaxPagesLimit             ParamName = "fuse-max-pages-limit"
	TransparentHugePages      ParamName = "transparent-hugepages"
	ReadAheadKb               ParamName = "read_ahead_kb"
	MaxBackgroundRequests     ParamName = "fuse-max-background-requests"
	CongestionWindowThreshold ParamName = "fuse-congestion-window-threshold"
)

// KernelParam represents an individual parameter setting.
type KernelParam struct {
	Name  ParamName `json:"name"`
	Value string    `json:"value"`
}

// KernelParamsConfig acts as the primary container for kernel settings.
type KernelParamsConfig struct {
	// Version is mandatory for cross-repo synchronization.
	// The consumer MUST validate this version before processing.
	Version    string        `json:"version"`
	RequestID  string        `json:"request_id"`
	Timestamp  string        `json:"timestamp"` // Format: 2026-01-12T16:23:05.636831Z
	Parameters []KernelParam `json:"parameters"`
}

// NewKernelParamsConfig initializes a new configuration container with an internal UUID.
func NewKernelParamsConfig() *KernelParamsConfig {
	return &KernelParamsConfig{
		Version:    CurrentContractVersion,
		RequestID:  uuid.NewString(),
		Timestamp:  time.Now().Format(TimeFormat),
		Parameters: make([]KernelParam, 0),
	}
}
