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

package main

import "time"

// Storage for all flags in the flag set returned by newFlagSet.
type flags struct {
	Help                               bool
	MountOptions                       map[string]string
	Uid                                int64
	Gid                                int64
	FileMode                           uint
	DirMode                            uint
	TempDir                            string
	TempDirLimit                       int64
	GCSChunkSize                       uint64
	ImplicitDirs                       bool
	StatCacheTTL                       time.Duration
	TypeCacheTTL                       time.Duration
	OpRateLimitHz                      float64
	EgressBandwidthLimitBytesPerSecond float64
}
